package agent

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/google/meetnotes"
)

// SyncGoogleMeetNotes pulls new notes from Gmail/Drive and ingests them.
func (a *AssistantAgent) SyncGoogleMeetNotes(ctx context.Context) (int, error) {
	if a.storage == nil {
		return 0, fmt.Errorf("assistant storage not available")
	}
	svc := meetnotes.NewService(a.storage.BaseDir())
	if !svc.IsConnected() {
		return 0, fmt.Errorf("google account not connected")
	}

	opts := meetnotes.SyncOptions{
		SenderEmail:  a.config.GoogleSenderEmail,
		BackfillDays: a.config.GoogleBackfillDays,
		MaxMessages:  50,
	}
	inputs, err := svc.Sync(ctx, opts)
	if err != nil {
		return 0, err
	}

	var ingested []*MeetingNote
	for _, in := range inputs {
		if note := a.ingestMeetingContent(ctx, in, false); note != nil {
			ingested = append(ingested, note)
		}
	}
	if len(ingested) > 0 {
		a.sendMeetingNotesBatchNotification(ctx, ingested)
	}
	log.Printf("[Assistant] Google meet notes sync: %d new notes", len(ingested))
	return len(ingested), nil
}

func (a *AssistantAgent) ingestMeetingContent(ctx context.Context, in meetnotes.IngestInput, notify bool) *MeetingNote {
	content := in.FullContent
	summary, attendees, actionItems, topics, err := a.analyzeMeetingContent(content)
	if err != nil {
		log.Printf("AI analysis failed, using basic parsing: %v", err)
		summary = a.extractBasicSummary(content)
		attendees = a.extractBasicAttendees(content)
		actionItems = a.extractBasicActionItems(content)
		topics = a.extractBasicTopics(content)
	}

	googleDocLink := in.GoogleDocLink
	if googleDocLink == "" && in.GoogleDocID != "" {
		googleDocLink = meetnotes.DocLink(in.GoogleDocID)
	}

	meetingNote := &MeetingNote{
		Source:         "google",
		GmailMessageID: in.GmailMessageID,
		GoogleDocID:    in.GoogleDocID,
		MeetingDate:    in.MeetingDate,
		Title:          in.Title,
		Attendees:      attendees,
		Summary:        summary,
		ActionItems:    actionItems,
		Topics:         topics,
		GoogleDocLink:  googleDocLink,
		FullContent:    content,
		IngestedAt:     time.Now(),
		CreatedBy:      a.Info.Name,
	}

	if a.storage != nil {
		if err := a.storage.SaveMeetingNote(meetingNote); err != nil {
			log.Printf("[Assistant] Failed to save meeting note: %v", err)
			return nil
		}
	}

	if notify {
		a.sendMeetingNoteNotification(ctx, meetingNote)
	}
	return meetingNote
}

// EnsureGoogleMeetNotesSync starts the background sync loop when connected (idempotent).
func (a *AssistantAgent) EnsureGoogleMeetNotesSync(ctx context.Context) {
	a.ensureGoogleMeetNotesSync(ctx)
}

func (a *AssistantAgent) ensureGoogleMeetNotesSync(ctx context.Context) {
	if !a.config.GoogleMeetNotesEnabled || a.storage == nil {
		return
	}
	svc := meetnotes.NewService(a.storage.BaseDir())
	if !svc.IsConnected() {
		if meetnotes.OAuthConfigured(a.storage.BaseDir()) {
			log.Printf("[Assistant] Google meet notes: connect via Settings to enable sync")
		} else {
			log.Printf("[Assistant] Google meet notes: set NEURAL_JUNKIE_GOOGLE_OAUTH_CLIENT_ID/SECRET on the hub")
		}
		return
	}
	a.googleSyncMu.Lock()
	defer a.googleSyncMu.Unlock()
	if a.googleSyncStarted {
		return
	}
	a.googleSyncStarted = true
	log.Printf("[Assistant] Starting Google meet notes sync (every %d min)", a.config.GoogleSyncIntervalMinutes)
	go a.runGoogleMeetNotesSync(ctx)
}

func (a *AssistantAgent) runGoogleMeetNotesSync(ctx context.Context) {
	if a.storage == nil {
		return
	}
	svc := meetnotes.NewService(a.storage.BaseDir())
	if !svc.IsConnected() {
		log.Printf("[Assistant] Google meet notes sync skipped: not connected")
		return
	}

	interval := time.Duration(a.config.GoogleSyncIntervalMinutes) * time.Minute
	if interval < time.Minute {
		interval = 15 * time.Minute
	}

	var syncMu sync.Mutex
	doSync := func() {
		syncMu.Lock()
		defer syncMu.Unlock()
		if _, err := a.SyncGoogleMeetNotes(ctx); err != nil {
			log.Printf("[Assistant] Google meet notes sync error: %v", err)
		}
	}

	doSync()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			doSync()
		case <-a.stopGoogleSync:
			return
		case <-ctx.Done():
			return
		}
	}
}

// GoogleMeetNotesStatus returns connection and sync metadata for the API.
func (a *AssistantAgent) GoogleMeetNotesStatus(ctx context.Context) (connected bool, email string, lastSync *time.Time, notesCount int, oauthConfigured bool, err error) {
	oauthConfigured = meetnotes.OAuthConfigured(a.storage.BaseDir())
	if a.storage == nil {
		return false, "", nil, 0, oauthConfigured, nil
	}
	svc := meetnotes.NewService(a.storage.BaseDir())
	connected = svc.IsConnected()
	lastSync = svc.StateStore.LastSyncAt()
	if connected {
		email, _ = svc.TokenStore.ConnectedEmail(ctx)
	}
	notes, loadErr := a.storage.LoadMeetingNotes()
	if loadErr == nil {
		notesCount = len(notes)
	}
	return connected, email, lastSync, notesCount, oauthConfigured, nil
}

// DisconnectGoogle clears Google OAuth tokens and sync state.
func (a *AssistantAgent) DisconnectGoogle() error {
	if a.storage == nil {
		return fmt.Errorf("assistant storage not available")
	}
	return (&meetnotes.TokenStore{BaseDir: a.storage.BaseDir()}).ClearToken()
}

// MeetNotesBaseDir returns the assistant storage path for Google meetnotes.
func (a *AssistantAgent) MeetNotesBaseDir() string {
	if a.storage == nil {
		return ""
	}
	return a.storage.BaseDir()
}
