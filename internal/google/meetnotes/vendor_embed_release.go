//go:build googlevendor

package meetnotes

import _ "embed"

//go:embed vendor/oauth.json
var vendorOAuthJSON []byte
