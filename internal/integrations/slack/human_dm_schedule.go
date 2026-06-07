package slack

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// WorkDayHours is a work window for one weekday (0=Monday … 6=Sunday).
type WorkDayHours struct {
	Weekday int    `json:"weekday"`
	Start   string `json:"start"` // HH:MM
	End     string `json:"end"`   // HH:MM
}

// HumanDMAwayConfig controls opt-in monitoring of the owner's human Slack DMs.
type HumanDMAwayConfig struct {
	Enabled          bool           `json:"enabled"`
	AwayEnabled      bool           `json:"away_enabled"`
	ScheduleEnabled  bool           `json:"schedule_enabled"`
	ScheduleTimezone string         `json:"schedule_timezone,omitempty"`
	WorkHours        []WorkDayHours `json:"work_hours,omitempty"`
	ReplyPrefix      string         `json:"reply_prefix,omitempty"`
	UserTokenSet     bool           `json:"user_token_set,omitempty"` // API only, not persisted
	MonitoringStatus string         `json:"monitoring_status,omitempty"` // API only
}

// DefaultWorkHours returns Mon–Fri 09:00–17:00.
func DefaultWorkHours() []WorkDayHours {
	hours := make([]WorkDayHours, 0, 5)
	for d := 0; d <= 4; d++ {
		hours = append(hours, WorkDayHours{Weekday: d, Start: "09:00", End: "17:00"})
	}
	return hours
}

func normalizeHumanDMAway(cfg *HumanDMAwayConfig) {
	if cfg == nil {
		return
	}
	if cfg.ScheduleTimezone == "" {
		cfg.ScheduleTimezone = "America/Los_Angeles"
	}
	if len(cfg.WorkHours) == 0 {
		cfg.WorkHours = DefaultWorkHours()
	}
	if strings.TrimSpace(cfg.ReplyPrefix) == "" {
		cfg.ReplyPrefix = "Assistant (for %s)"
	}
}

// ShouldMonitorHumanDMs reports whether away/schedule should poll human DMs for agent auto-reply.
func ShouldMonitorHumanDMs(inbox InboxConfig, userTokenSet bool, now time.Time) bool {
	if !humanDMPollingReady(inbox, userTokenSet) {
		return false
	}
	h := inbox.HumanDMAway
	if h.AwayEnabled {
		return true
	}
	if !h.ScheduleEnabled {
		return false
	}
	normalizeHumanDMAway(&h)
	return isOutsideWorkHours(now, h.ScheduleTimezone, h.WorkHours)
}

// ShouldPollHumanDMs reports whether the human-DM poller should run (away/schedule or forward mode).
func ShouldPollHumanDMs(inbox InboxConfig, userTokenSet bool, now time.Time) bool {
	if !humanDMPollingReady(inbox, userTokenSet) {
		return false
	}
	if ShouldMonitorHumanDMs(inbox, userTokenSet, now) {
		return true
	}
	return inbox.ForwardEnabled
}

// ShouldAutoReplyHumanDMs reports whether ingested human DMs should route to the inbox agent.
func ShouldAutoReplyHumanDMs(inbox InboxConfig, userTokenSet bool, now time.Time) bool {
	return ShouldMonitorHumanDMs(inbox, userTokenSet, now)
}

func humanDMPollingReady(inbox InboxConfig, userTokenSet bool) bool {
	if !inbox.Enabled || inbox.AgentID == "" || inbox.OwnerSlackUserID == "" {
		return false
	}
	h := inbox.HumanDMAway
	return h.Enabled && userTokenSet
}

func isOutsideWorkHours(now time.Time, tz string, hours []WorkDayHours) bool {
	loc := time.Local
	if tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}
	local := now.In(loc)
	weekday := (int(local.Weekday()) + 6) % 7 // Mon=0 … Sun=6
	var day *WorkDayHours
	for i := range hours {
		if hours[i].Weekday == weekday {
			day = &hours[i]
			break
		}
	}
	if day == nil {
		return true
	}
	start, ok1 := parseClock(day.Start)
	end, ok2 := parseClock(day.End)
	if !ok1 || !ok2 {
		return true
	}
	cur := local.Hour()*60 + local.Minute()
	return cur < start || cur >= end
}

func parseClock(s string) (minutes int, ok bool) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// HumanDMReplyPrefix returns the formatted reply prefix for outbound human DM posts.
func HumanDMReplyPrefix(cfg HumanDMAwayConfig, ownerName string) string {
	normalizeHumanDMAway(&cfg)
	name := strings.TrimSpace(ownerName)
	if name == "" {
		name = "you"
	}
	return fmt.Sprintf(cfg.ReplyPrefix, name) + ": "
}

// HumanDMMonitoringStatus returns a UI-friendly monitoring state label.
func HumanDMMonitoringStatus(inbox InboxConfig, userTokenSet bool, now time.Time) string {
	h := inbox.HumanDMAway
	if !h.Enabled {
		return "disabled"
	}
	if !userTokenSet {
		return "not_authorized"
	}
	if !inbox.Enabled || inbox.AgentID == "" {
		return "inbox_not_ready"
	}
	if ShouldMonitorHumanDMs(inbox, userTokenSet, now) {
		return "monitoring_active"
	}
	if inbox.ForwardEnabled {
		return "forward_active"
	}
	if h.AwayEnabled {
		return "away_off"
	}
	if h.ScheduleEnabled {
		return "inside_work_hours"
	}
	return "away_off"
}
