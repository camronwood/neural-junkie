package meetnotes

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func newGmailService(ctx context.Context, ts oauth2.TokenSource) (*gmail.Service, error) {
	client := oauth2.NewClient(ctx, ts)
	return gmail.NewService(ctx, option.WithHTTPClient(client))
}

func newDriveService(ctx context.Context, ts oauth2.TokenSource) (*drive.Service, error) {
	client := oauth2.NewClient(ctx, ts)
	return drive.NewService(ctx, option.WithHTTPClient(client))
}

// exportDocText exports a Google Doc as plain text.
func exportDocText(ctx context.Context, driveSvc *drive.Service, docID string) (string, error) {
	resp, err := driveSvc.Files.Export(docID, "text/plain").Download()
	if err != nil {
		return "", fmt.Errorf("export doc %s: %w", docID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("export doc %s: status %s", docID, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
