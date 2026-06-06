package agent

// CLIInstallSpec describes how to install a CLI tool from the app.
type CLIInstallSpec struct {
	Method  string   `json:"method"`            // npm, curl, brew, pip
	Command string   `json:"command"`           // shell command to run
	Prereqs []string `json:"prereqs,omitempty"` // binaries required on PATH first
}

// CLIAuthSpec describes how to authenticate a CLI tool.
type CLIAuthSpec struct {
	Method          string   `json:"method"`                     // api_key, cli_login
	EnvVars         []string `json:"env_vars,omitempty"`         // e.g. CURSOR_API_KEY
	LoginCommand    []string `json:"login_command,omitempty"`    // e.g. ["agent", "login"]
	ProbeCommand    []string `json:"probe_command,omitempty"`    // e.g. ["agent", "status"]
	CredentialPaths []string `json:"credential_paths,omitempty"` // optional marker files when authed
}

// FeaturedCLITypes are shown prominently in setup and settings UI.
var FeaturedCLITypes = []string{"cursor", "claude", "gemini"}

// HasInstallSpec reports whether one-click install metadata is configured.
func (c CLIAgentConfig) HasInstallSpec() bool {
	return c.Install != nil && c.Install.Command != ""
}

// HasAuthSpec reports whether auth metadata is configured.
func (c CLIAgentConfig) HasAuthSpec() bool {
	return c.Auth != nil
}

// IsFeaturedCLI reports whether this type is in the featured list.
func IsFeaturedCLI(cliType string) bool {
	for _, t := range FeaturedCLITypes {
		if t == cliType {
			return true
		}
	}
	return false
}
