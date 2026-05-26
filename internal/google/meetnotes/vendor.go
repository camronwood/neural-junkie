package meetnotes

import (
	"encoding/json"
	"strings"
)

// bundledVendorCredentials is the shape of vendor/oauth.json and .example.
type bundledVendorCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url,omitempty"`
}

func oauthFromBundledVendor() (*AppOAuthCredentials, bool) {
	if len(vendorOAuthJSON) == 0 {
		return nil, false
	}
	var v bundledVendorCredentials
	if err := json.Unmarshal(vendorOAuthJSON, &v); err != nil {
		return nil, false
	}
	clientID := strings.TrimSpace(v.ClientID)
	secret := strings.TrimSpace(v.ClientSecret)
	if clientID == "" || secret == "" {
		return nil, false
	}
	if strings.Contains(clientID, "YOUR_") || strings.Contains(secret, "YOUR_") {
		return nil, false
	}
	redirectURL := strings.TrimSpace(v.RedirectURL)
	if redirectURL == "" {
		redirectURL = defaultRedirectURL
	}
	return &AppOAuthCredentials{
		ClientID:     clientID,
		ClientSecret: secret,
		RedirectURL:  redirectURL,
	}, true
}
