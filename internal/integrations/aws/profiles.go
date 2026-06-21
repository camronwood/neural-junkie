package awsprofiles

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ListProfiles reads named profiles from ~/.aws/config and [default] from credentials.
func ListProfiles() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var out []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if err := scanConfigProfiles(filepath.Join(home, ".aws", "config"), add); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := scanCredentialsDefault(filepath.Join(home, ".aws", "credentials"), add); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return out, nil
}

func scanConfigProfiles(path string, add func(string)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
			continue
		}
		inner := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
		if strings.HasPrefix(inner, "profile ") {
			add(strings.TrimPrefix(inner, "profile "))
		} else if inner == "default" {
			add("default")
		}
	}
	return sc.Err()
}

func scanCredentialsDefault(path string, add func(string)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "[default]" {
			add("default")
			break
		}
	}
	return sc.Err()
}
