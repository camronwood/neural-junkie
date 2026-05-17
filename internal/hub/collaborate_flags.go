package hub

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

// parseCollaborateLeadFlags reads optional discussion limits from the start of
// the command tail (parts[1:]). Flags must appear before @mentions and the
// description. Unrecognized tokens stop flag parsing so @-mentions are never
// consumed as flag values.
//
// Supported: --rounds N, -rounds N, --messages N, --max-messages N, -messages N
func parseCollaborateLeadFlags(parts []string) (collaboration.DiscussionConfig, []string, string) {
	var cfg collaboration.DiscussionConfig
	if len(parts) < 2 {
		return cfg, nil, ""
	}
	i := 1
	for i < len(parts) {
		raw := parts[i]
		if !strings.HasPrefix(raw, "-") {
			break
		}
		key := stripCollaborateFlagPrefix(strings.ToLower(raw))
		switch key {
		case "rounds":
			if i+1 >= len(parts) {
				return cfg, nil, "❌ `--rounds` needs a number (e.g. `--rounds 5`)."
			}
			n, err := strconv.Atoi(parts[i+1])
			if err != nil {
				return cfg, nil, fmt.Sprintf("❌ Invalid `--rounds` value: %q", parts[i+1])
			}
			cfg.MaxRounds = n
			i += 2
		case "messages", "max-messages":
			if i+1 >= len(parts) {
				return cfg, nil, "❌ `--messages` needs a number (e.g. `--messages 30`)."
			}
			n, err := strconv.Atoi(parts[i+1])
			if err != nil {
				return cfg, nil, fmt.Sprintf("❌ Invalid `--messages` value: %q", parts[i+1])
			}
			cfg.MaxTotalMessages = n
			i += 2
		default:
			return cfg, nil, fmt.Sprintf("❌ Unknown option %q. Use `--rounds` and/or `--messages` before @mentions.", raw)
		}
	}
	return cfg, parts[i:], ""
}

func stripCollaborateFlagPrefix(s string) string {
	s = strings.TrimPrefix(s, "--")
	s = strings.TrimPrefix(s, "-")
	return s
}

// parseCollabExtendArgs parses `/collab-extend <id-prefix> [--rounds N] [--messages M]`.
func parseCollabExtendArgs(parts []string) (id string, extraRounds, extraMessages int, errMsg string) {
	if len(parts) < 2 {
		return "", 0, 0, "❌ Usage: /collab-extend <collab-id> [--rounds N] [--messages M]"
	}
	id = parts[1]
	i := 2
	for i < len(parts) {
		raw := parts[i]
		if !strings.HasPrefix(raw, "-") {
			// Accept shorthand: `/collab-extend ec2cdef8 1` → one extra round,
			// `/collab-extend ec2cdef8 1 4` → rounds then messages.
			n, err := strconv.Atoi(raw)
			if err != nil || n <= 0 {
				return "", 0, 0, fmt.Sprintf("❌ Unexpected argument %q after collab id. Use flags: --rounds N --messages M (or bare numbers: id rounds [messages]).", raw)
			}
			if extraRounds == 0 {
				extraRounds = n
			} else if extraMessages == 0 {
				extraMessages = n
			} else {
				return "", 0, 0, fmt.Sprintf("❌ Too many numeric arguments after collab id. Use --rounds and --messages flags.")
			}
			i++
			continue
		}
		key := stripCollaborateFlagPrefix(strings.ToLower(raw))
		switch key {
		case "rounds":
			if i+1 >= len(parts) {
				return "", 0, 0, "❌ `--rounds` needs a positive number."
			}
			n, err := strconv.Atoi(parts[i+1])
			if err != nil || n <= 0 {
				return "", 0, 0, fmt.Sprintf("❌ Invalid `--rounds` value: %q", parts[i+1])
			}
			extraRounds += n
			i += 2
		case "messages", "max-messages":
			if i+1 >= len(parts) {
				return "", 0, 0, "❌ `--messages` needs a positive number."
			}
			n, err := strconv.Atoi(parts[i+1])
			if err != nil || n <= 0 {
				return "", 0, 0, fmt.Sprintf("❌ Invalid `--messages` value: %q", parts[i+1])
			}
			extraMessages += n
			i += 2
		default:
			return "", 0, 0, fmt.Sprintf("❌ Unknown option %q. Use `--rounds` and/or `--messages`.", raw)
		}
	}
	return id, extraRounds, extraMessages, ""
}
