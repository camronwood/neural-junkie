package packs

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed capability_tokens.json
var capabilityTokensJSON []byte

type capabilityTokensFile struct {
	Platform           []string `json:"platform"`
	OfficialDomain     []string `json:"official_domain"`
	PackLocalExamples  []string `json:"pack_local_examples"`
}

var (
	capabilityTokensOnce sync.Once
	capabilityTokensData capabilityTokensFile
)

func loadCapabilityTokens() capabilityTokensFile {
	capabilityTokensOnce.Do(func() {
		if err := json.Unmarshal(capabilityTokensJSON, &capabilityTokensData); err != nil {
			panic("packs: invalid capability_tokens.json: " + err.Error())
		}
	})
	return capabilityTokensData
}

func platformCapabilityTokensFromJSON() []string {
	return append([]string{}, loadCapabilityTokens().Platform...)
}

func officialDomainCapabilityTokensFromJSON() []string {
	return append([]string{}, loadCapabilityTokens().OfficialDomain...)
}

// PackLocalExampleCapabilityTokens are documented pack-local short ids (not NJ KnownCapabilityTokens).
func PackLocalExampleCapabilityTokens() []string {
	return append([]string{}, loadCapabilityTokens().PackLocalExamples...)
}
