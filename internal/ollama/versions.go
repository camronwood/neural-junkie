package ollama

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// RecommendedOllamaVersion is the NJ-pinned runtime target (keep in sync with scripts/fetch-ollama.sh).
const RecommendedOllamaVersion = "0.32.0"

// MinOllamaVersion is the floor for models that need modern GGUF support (e.g. Bonsai Q1_0).
const MinOllamaVersion = "0.30.0"

// RecommendedOllamaTag is the GitHub release tag for downloads (v-prefixed).
const RecommendedOllamaTag = "v" + RecommendedOllamaVersion

var (
	semverRe     = regexp.MustCompile(`(?i)\b(\d+)\.(\d+)\.(\d+)\b`)
	clientVerRe  = regexp.MustCompile(`(?i)client version is\s+(\d+\.\d+\.\d+)`)
	ollamaVerRe  = regexp.MustCompile(`(?i)ollama version is\s+(\d+\.\d+\.\d+)`)
	apiVersionRe = regexp.MustCompile(`(?i)"version"\s*:\s*"([^"]+)"`)
)

// ParseOllamaVersion extracts a semver from `ollama --version` output or an API version string.
// Prefers "client version is X" (binary) over "ollama version is X" (often the remote server).
func ParseOllamaVersion(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if m := clientVerRe.FindStringSubmatch(raw); len(m) == 2 {
		return m[1], true
	}
	if m := ollamaVerRe.FindStringSubmatch(raw); len(m) == 2 {
		return m[1], true
	}
	trimmed := strings.TrimPrefix(strings.TrimSpace(raw), "v")
	trimmed = strings.TrimPrefix(trimmed, "V")
	if m := semverRe.FindStringSubmatch(trimmed); len(m) == 4 {
		return fmt.Sprintf("%s.%s.%s", m[1], m[2], m[3]), true
	}
	if m := semverRe.FindStringSubmatch(raw); len(m) == 4 {
		return fmt.Sprintf("%s.%s.%s", m[1], m[2], m[3]), true
	}
	return "", false
}

// ParseAPIVersion extracts version from GET /api/version JSON body.
func ParseAPIVersion(body string) (string, bool) {
	if m := apiVersionRe.FindStringSubmatch(body); len(m) == 2 {
		return ParseOllamaVersion(m[1])
	}
	return ParseOllamaVersion(body)
}

type semver struct {
	major, minor, patch int
}

func parseSemverParts(v string) (semver, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	m := semverRe.FindStringSubmatch(v)
	if len(m) != 4 {
		return semver{}, false
	}
	maj, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	pat, _ := strconv.Atoi(m[3])
	return semver{maj, min, pat}, true
}

// CompareSemver returns -1 if a<b, 0 if equal, 1 if a>b. Invalid versions compare as less than valid.
func CompareSemver(a, b string) int {
	sa, oka := parseSemverParts(a)
	sb, okb := parseSemverParts(b)
	if !oka && !okb {
		return 0
	}
	if !oka {
		return -1
	}
	if !okb {
		return 1
	}
	if sa.major != sb.major {
		if sa.major < sb.major {
			return -1
		}
		return 1
	}
	if sa.minor != sb.minor {
		if sa.minor < sb.minor {
			return -1
		}
		return 1
	}
	if sa.patch != sb.patch {
		if sa.patch < sb.patch {
			return -1
		}
		return 1
	}
	return 0
}

// NeedsUpdate reports whether installed is older than RecommendedOllamaVersion.
func NeedsUpdate(installed string) bool {
	v, ok := ParseOllamaVersion(installed)
	if !ok {
		return installed != "" // unknown but present — surface update
	}
	return CompareSemver(v, RecommendedOllamaVersion) < 0
}

// MeetsMinimum reports whether installed is at least MinOllamaVersion.
func MeetsMinimum(installed string) bool {
	v, ok := ParseOllamaVersion(installed)
	if !ok {
		return false
	}
	return CompareSemver(v, MinOllamaVersion) >= 0
}

// MeetsVersion reports whether installed is at least required (empty required => true).
func MeetsVersion(installed, required string) bool {
	required = strings.TrimSpace(required)
	if required == "" {
		return true
	}
	v, ok := ParseOllamaVersion(installed)
	if !ok {
		return false
	}
	req, ok := ParseOllamaVersion(required)
	if !ok {
		req = required
	}
	return CompareSemver(v, req) >= 0
}
