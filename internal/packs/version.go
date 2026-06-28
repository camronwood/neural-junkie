package packs

import (
	"strings"

	"golang.org/x/mod/semver"
)

// NormalizeVersion returns a valid semver string with v prefix, or "" if invalid.
func NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return ""
	}
	return v
}

// VersionLess reports whether semver a is strictly less than b.
func VersionLess(a, b string) bool {
	na, nb := NormalizeVersion(a), NormalizeVersion(b)
	if na == "" || nb == "" {
		return false
	}
	return semver.Compare(na, nb) < 0
}

// UpdateAvailable reports whether catalog is newer than installed.
func UpdateAvailable(installed, catalog string) bool {
	ni, nc := NormalizeVersion(installed), NormalizeVersion(catalog)
	if nc == "" {
		return false
	}
	if ni == "" {
		return true
	}
	return semver.Compare(ni, nc) < 0
}
