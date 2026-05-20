package meetnotes

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/mail"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
)

// IngestInput is one meeting note ready for assistant ingestion.
type IngestInput struct {
	Title           string
	MeetingDate     time.Time
	GoogleDocID     string
	GoogleDocLink   string
	GmailMessageID  string
	FullContent     string
}

// SyncOptions configures a sync run.
type SyncOptions struct {
	SenderEmail   string
	BackfillDays  int
	MaxMessages   int // 0 = no cap per run
}

// DefaultSyncOptions returns sensible defaults.
func DefaultSyncOptions() SyncOptions {
	return SyncOptions{
		SenderEmail:  "gemini-notes@google.com",
		BackfillDays: 90,
		MaxMessages:  50,
	}
}

// Service syncs Gemini meeting notes from Gmail + Drive.
type Service struct {
	TokenStore *TokenStore
	StateStore *StateStore
}

// NewService creates a sync service for the given assistant base directory.
func NewService(baseDir string) *Service {
	return &Service{
		TokenStore: &TokenStore{BaseDir: baseDir},
		StateStore: &StateStore{BaseDir: baseDir},
	}
}

// IsConnected reports whether OAuth tokens exist.
func (s *Service) IsConnected() bool {
	return s.TokenStore.HasValidToken()
}

// Sync fetches new Gemini note emails and exports linked Google Docs.
func (s *Service) Sync(ctx context.Context, opts SyncOptions) ([]IngestInput, error) {
	if opts.SenderEmail == "" {
		opts.SenderEmail = DefaultSyncOptions().SenderEmail
	}
	if opts.BackfillDays <= 0 {
		opts.BackfillDays = 90
	}

	ts, err := s.TokenStore.ClientSource(ctx)
	if err != nil {
		return nil, err
	}

	gmailSvc, err := newGmailService(ctx, ts)
	if err != nil {
		return nil, err
	}
	driveSvc, err := newDriveService(ctx, ts)
	if err != nil {
		return nil, err
	}

	state, err := s.StateStore.Load()
	if err != nil {
		return nil, err
	}

	after := time.Now().AddDate(0, 0, -opts.BackfillDays).Unix()
	query := fmt.Sprintf("from:%s after:%d", opts.SenderEmail, after)

	listCall := gmailSvc.Users.Messages.List("me").Q(query).MaxResults(100)
	resp, err := listCall.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("list gmail messages: %w", err)
	}

	var inputs []IngestInput
	processed := 0
	for _, ref := range resp.Messages {
		if opts.MaxMessages > 0 && processed >= opts.MaxMessages {
			break
		}
		if state.ProcessedMessageIDs[ref.Id] {
			continue
		}

		msg, err := gmailSvc.Users.Messages.Get("me", ref.Id).Format("full").Context(ctx).Do()
		if err != nil {
			log.Printf("[meetnotes] get message %s: %v", ref.Id, err)
			continue
		}

		subject, date, body := parseGmailMessage(msg)
		docID := ExtractDocID(body)
		if docID == "" {
			log.Printf("[meetnotes] skip message %s: no Google Doc link in body", ref.Id)
			state.ProcessedMessageIDs[ref.Id] = true
			continue
		}

		text, err := exportDocText(ctx, driveSvc, docID)
		if err != nil {
			log.Printf("[meetnotes] export doc %s for message %s: %v", docID, ref.Id, err)
			continue
		}
		if strings.TrimSpace(text) == "" {
			log.Printf("[meetnotes] skip message %s: empty doc export", ref.Id)
			state.ProcessedMessageIDs[ref.Id] = true
			continue
		}

		meetingDate := date
		if meetingDate.IsZero() {
			meetingDate = time.Now()
		}
		title := subject
		if title == "" {
			title = "Meeting notes"
		}

		inputs = append(inputs, IngestInput{
			Title:          title,
			MeetingDate:    meetingDate,
			GoogleDocID:    docID,
			GoogleDocLink:  DocLink(docID),
			GmailMessageID: ref.Id,
			FullContent:    text,
		})

		state.ProcessedMessageIDs[ref.Id] = true
		processed++
	}

	if err := s.StateStore.Save(state); err != nil {
		return inputs, err
	}
	return inputs, nil
}

func parseGmailMessage(msg *gmail.Message) (subject string, date time.Time, body string) {
	var sb strings.Builder
	collectParts(msg.Payload, &sb)
	body = sb.String()

	for _, h := range msg.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "subject":
			subject = h.Value
		case "date":
			if t, err := mail.ParseDate(h.Value); err == nil {
				date = t
			}
		}
	}
	if date.IsZero() && msg.InternalDate > 0 {
		date = time.UnixMilli(msg.InternalDate)
	}
	return subject, date, body
}

func collectParts(part *gmail.MessagePart, sb *strings.Builder) {
	if part == nil {
		return
	}
	if part.Body != nil && part.Body.Data != "" {
		if decoded, err := base64.URLEncoding.DecodeString(part.Body.Data); err == nil {
			sb.Write(decoded)
		}
	}
	for _, child := range part.Parts {
		collectParts(child, sb)
	}
}
