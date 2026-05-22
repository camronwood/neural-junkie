package slack

import (
	"log"
	"strings"

	slackapi "github.com/slack-go/slack"
)

type cachedSlackUser struct {
	Display string
	Handle  string
}

// resolveInboundUserIdentity fills in.UserName and in.SlackUsername from cache or users.info.
func (b *Bridge) resolveInboundUserIdentity(in *InboundInput) {
	if in == nil || in.UserID == "" {
		if in != nil && in.UserName == "" {
			in.UserName = "Slack User"
		}
		return
	}
	if v, ok := b.userNames.Load(in.UserID); ok {
		if c, ok := v.(cachedSlackUser); ok {
			in.SlackUsername = c.Handle
			in.UserName = FormatSlackSenderLabel(c.Display, c.Handle)
		}
	} else if u, err := b.api.GetUserInfo(in.UserID); err == nil && u != nil {
		display, handle := DisplayNameFromUser(u)
		b.userNames.Store(in.UserID, cachedSlackUser{Display: display, Handle: handle})
		in.SlackUsername = handle
		in.UserName = FormatSlackSenderLabel(display, handle)
	} else if InboundDebugEnabled() {
		log.Printf("[slack] users.info %s: %v (need users:read scope?)", in.UserID, err)
	}
	if in.UserName == "" {
		in.UserName = "Slack User"
		if in.SlackUsername != "" {
			in.UserName = "@" + in.SlackUsername
		}
	}
}

// DisplayNameFromUser picks the best human-readable label for NJ (display name, real name, @handle).
func DisplayNameFromUser(u *slackapi.User) (display, handle string) {
	if u == nil {
		return "", ""
	}
	handle = strings.TrimSpace(u.Name)
	p := u.Profile
	candidates := []string{
		strings.TrimSpace(p.DisplayName),
		strings.TrimSpace(p.DisplayNameNormalized),
		strings.TrimSpace(p.RealName),
		strings.TrimSpace(p.RealNameNormalized),
		strings.TrimSpace(u.RealName),
	}
	for _, c := range candidates {
		if c != "" {
			display = c
			break
		}
	}
	if display == "" {
		display = strings.TrimSpace(strings.TrimSpace(p.FirstName) + " " + strings.TrimSpace(p.LastName))
	}
	if display == "" {
		display = handle
	}
	return display, handle
}

// FormatSlackSenderLabel returns the name shown in NJ; appends @handle when it differs from display.
func FormatSlackSenderLabel(display, handle string) string {
	display = strings.TrimSpace(display)
	handle = strings.TrimSpace(handle)
	if display == "" && handle == "" {
		return ""
	}
	if display == "" {
		return "@" + handle
	}
	if handle == "" || strings.EqualFold(display, handle) {
		return display
	}
	return display + " (@" + handle + ")"
}
