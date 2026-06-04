package phoeniximport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type listDocumentsOptions struct {
	limit int
	sort  string
	query map[string]any
}

type timClient struct {
	http            *http.Client
	profile         environmentProfile
	creds           storedCredentials
	credentialsPath string
}

func newTimClient(ctx context.Context, settings Settings) (*timClient, error) {
	profile, err := profileForEnvironment(settings.EnvironmentOrDefault())
	if err != nil {
		return nil, err
	}
	creds, credPath, err := ensureCredentials(ctx, settings)
	if err != nil {
		return nil, err
	}
	return &timClient{
		http: &http.Client{Timeout: 4 * time.Minute},
		profile: profile,
		creds: creds,
		credentialsPath: credPath,
	}, nil
}

func (c *timClient) bearer() string {
	return "Bearer " + c.creds.AccessToken
}

func (c *timClient) base() string {
	return strings.TrimSuffix(c.profile.APIBase, "/")
}

func (c *timClient) whoami() string {
	return identityFromToken(c.creds.AccessToken)
}

func (c *timClient) getDocument(ctx context.Context, collection, id string) (json.RawMessage, error) {
	u := fmt.Sprintf("%s/%s/%s", c.base(), collection, id)
	return c.getJSON(ctx, u)
}

func (c *timClient) listDocuments(ctx context.Context, collection string, opts listDocumentsOptions) (json.RawMessage, error) {
	u := fmt.Sprintf("%s/%s", c.base(), collection)
	params := url.Values{}
	if opts.limit > 0 {
		params.Set("limit", strconv.Itoa(opts.limit))
	}
	if s := strings.TrimSpace(opts.sort); s != "" {
		params.Set("sort", s)
	}
	if len(opts.query) > 0 {
		b, err := json.Marshal(opts.query)
		if err != nil {
			return nil, err
		}
		params.Set("query", string(b))
	}
	if enc := params.Encode(); enc != "" {
		u += "?" + enc
	}
	return c.getJSON(ctx, u)
}

func (c *timClient) listAttachments(ctx context.Context, collection, id string) ([]string, error) {
	u := fmt.Sprintf("%s/%s/%s/attachments", c.base(), collection, id)
	raw, err := c.getJSON(ctx, u)
	if err != nil {
		return nil, err
	}
	return parseAttachmentNames(raw), nil
}

func (c *timClient) downloadAttachment(ctx context.Context, collection, id, name, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	presignURL := fmt.Sprintf("%s/%s/%s/attachments/%s?presign=true", c.base(), collection, id, url.PathEscape(name))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, presignURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.bearer())
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("presign HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var presign struct {
		DownloadURL string `json:"downloadUrl"`
	}
	if err := json.Unmarshal(body, &presign); err != nil {
		return err
	}
	if strings.TrimSpace(presign.DownloadURL) == "" {
		return fmt.Errorf("presign response missing downloadUrl")
	}
	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, presign.DownloadURL, nil)
	if err != nil {
		return err
	}
	dlRes, err := c.http.Do(dlReq)
	if err != nil {
		return err
	}
	defer dlRes.Body.Close()
	if dlRes.StatusCode < 200 || dlRes.StatusCode >= 300 {
		b, _ := io.ReadAll(dlRes.Body)
		return fmt.Errorf("attachment download HTTP %d: %s", dlRes.StatusCode, strings.TrimSpace(string(b)))
	}
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, dlRes.Body)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func (c *timClient) getJSON(ctx context.Context, u string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.bearer())
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", u, err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.RawMessage(body), nil
}
