// Package capabilities loads benchmark-derived Ollama model rankings for task routing.
package capabilities

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TaskClass identifies a routing profile bucket.
type TaskClass string

const (
	TaskImplement      TaskClass = "implement"
	TaskChat           TaskClass = "chat"
	TaskCollabLight    TaskClass = "collab_light"
	TaskUtility        TaskClass = "utility"
	TaskAskMode        TaskClass = "ask_mode"
	TaskImplementHeavy TaskClass = "implement_heavy"
)

// ModelScore holds per-model benchmark metrics.
type ModelScore struct {
	ImplementPassRate float64          `json:"implement_pass_rate"`
	ChatPassRate      float64          `json:"chat_pass_rate"`
	OverallPassRate   float64          `json:"overall_pass_rate"`
	ParamsB           *float64         `json:"params_b"`
	Scenarios         map[string]bool  `json:"scenarios"`
}

// Profiles is the v1 capability routing artifact.
type Profiles struct {
	UpdatedAt    string                `json:"updated_at"`
	SourceRunID  string                `json:"source_run_id"`
	SourceSuite  string                `json:"source_suite"`
	TaskClasses  map[string][]string   `json:"task_classes"`
	ModelScores  map[string]ModelScore `json:"model_scores"`
}

// ParseProfiles decodes JSON capability profiles.
func ParseProfiles(data []byte) (*Profiles, error) {
	var p Profiles
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse capability profiles: %w", err)
	}
	if p.TaskClasses == nil {
		p.TaskClasses = map[string][]string{}
	}
	if p.ModelScores == nil {
		p.ModelScores = map[string]ModelScore{}
	}
	return &p, nil
}

// Tags returns ranked model tags for a task class.
func (p *Profiles) Tags(class TaskClass) []string {
	if p == nil {
		return nil
	}
	return append([]string(nil), p.TaskClasses[string(class)]...)
}

// RankIndex returns the rank position of tag in class (0 = best). -1 if absent.
func (p *Profiles) RankIndex(class TaskClass, tag string) int {
	tag = strings.TrimSpace(tag)
	if tag == "" || p == nil {
		return -1
	}
	for i, t := range p.Tags(class) {
		if t == tag {
			return i
		}
	}
	return -1
}

// Status returns read-only metadata for settings UI.
func (p *Profiles) Status() map[string]string {
	if p == nil {
		return map[string]string{}
	}
	return map[string]string{
		"updated_at":    p.UpdatedAt,
		"source_run_id": p.SourceRunID,
		"source_suite":  p.SourceSuite,
	}
}
