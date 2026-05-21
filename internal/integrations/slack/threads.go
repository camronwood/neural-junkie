package slack

import (
	"encoding/json"
	"os"
	"sync"
)

// ThreadMap links Slack thread_ts to Neural Junkie thread_id / reply_to.
type ThreadMap struct {
	mu       sync.RWMutex
	filePath string
	// slackChannelID -> slack parent ts -> nj thread root message ID
	roots map[string]map[string]string
	// nj thread ID -> slack thread_ts for replies
	njToSlack map[string]string
}

// NewThreadMap loads thread mappings from disk.
func NewThreadMap() (*ThreadMap, error) {
	p, err := threadsPath()
	if err != nil {
		return nil, err
	}
	t := &ThreadMap{
		filePath:  p,
		roots:     make(map[string]map[string]string),
		njToSlack: make(map[string]string),
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return t, nil
		}
		return nil, err
	}
	var payload struct {
		Roots     map[string]map[string]string `json:"roots"`
		NJToSlack map[string]string            `json:"nj_to_slack"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.Roots != nil {
		t.roots = payload.Roots
	}
	if payload.NJToSlack != nil {
		t.njToSlack = payload.NJToSlack
	}
	return t, nil
}

func (t *ThreadMap) save() error {
	t.mu.RLock()
	payload := struct {
		Roots     map[string]map[string]string `json:"roots"`
		NJToSlack map[string]string            `json:"nj_to_slack"`
	}{
		Roots:     t.roots,
		NJToSlack: t.njToSlack,
	}
	t.mu.RUnlock()
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	tmp := t.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, t.filePath)
}

// ResolveInbound returns nj thread_id, reply_to, isThreadReply for a Slack message.
func (t *ThreadMap) ResolveInbound(slackChannelID, slackTS, threadTS string) (threadID, replyTo string, isThread bool) {
	if threadTS == "" || threadTS == slackTS {
		return "", "", false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	ch := t.roots[slackChannelID]
	rootID := ""
	if ch != nil {
		rootID = ch[threadTS]
	}
	if rootID == "" {
		// First reply in thread: parent ts is the root.
		return threadTS, threadTS, true
	}
	return rootID, slackTS, true
}

// RegisterInboundRoot records a new thread root when the first reply arrives.
func (t *ThreadMap) RegisterInboundRoot(slackChannelID, slackParentTS, njRootID string) error {
	t.mu.Lock()
	if t.roots[slackChannelID] == nil {
		t.roots[slackChannelID] = make(map[string]string)
	}
	t.roots[slackChannelID][slackParentTS] = njRootID
	t.njToSlack[njRootID] = slackParentTS
	t.mu.Unlock()
	return t.save()
}

// SlackThreadTS returns the Slack thread_ts for posting a reply in an NJ thread.
func (t *ThreadMap) SlackThreadTS(njThreadID string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if ts, ok := t.njToSlack[njThreadID]; ok {
		return ts
	}
	return njThreadID
}

// RegisterOutbound records the Slack ts of an agent reply for future threading.
func (t *ThreadMap) RegisterOutbound(njThreadID, slackTS string) error {
	if njThreadID == "" || slackTS == "" {
		return nil
	}
	t.mu.Lock()
	t.njToSlack[njThreadID] = slackTS
	t.mu.Unlock()
	return t.save()
}
