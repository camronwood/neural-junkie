//go:build !slackvendor

package slack

import _ "embed"

//go:embed vendor/oauth.json.example
var vendorOAuthJSON []byte
