package slack

// OAuthV2AccessResponse is the Slack oauth.v2.access JSON payload.
type OAuthV2AccessResponse struct {
	OK          bool   `json:"ok"`
	Error       string `json:"error"`
	AccessToken string `json:"access_token"`
	BotUserID   string `json:"bot_user_id"`
	AuthedUser  struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
	} `json:"authed_user"`
	Team struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
}
