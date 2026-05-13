package hub

import (
	"runtime"
	"sort"
)

// HubMemoryReport returns lightweight hub + runtime stats for debugging memory pressure.
// Caller should treat this as diagnostic-only; it holds RLock for the duration of the scan.
func (h *Hub) HubMemoryReport() map[string]interface{} {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	h.mu.RLock()
	defer h.mu.RUnlock()

	type chRow struct {
		Name         string `json:"name"`
		Messages     int    `json:"messages"`
		ContentBytes int64  `json:"content_bytes"`
	}
	channels := make([]chRow, 0, len(h.messages))
	var totalMsgs int
	var totalContent int64
	for name, msgs := range h.messages {
		var b int64
		for _, m := range msgs {
			if m != nil {
				b += int64(len(m.Content))
			}
		}
		n := len(msgs)
		totalMsgs += n
		totalContent += b
		channels = append(channels, chRow{Name: name, Messages: n, ContentBytes: b})
	}
	sort.Slice(channels, func(i, j int) bool {
		if channels[i].ContentBytes != channels[j].ContentBytes {
			return channels[i].ContentBytes > channels[j].ContentBytes
		}
		return channels[i].Name < channels[j].Name
	})

	collabN := 0
	if h.collabManager != nil {
		collabN = h.collabManager.Len()
	}

	return map[string]interface{}{
		"go_alloc_mb":       ms.Alloc / (1024 * 1024),
		"go_heap_sys_mb":    ms.HeapSys / (1024 * 1024),
		"go_sys_mb":         ms.Sys / (1024 * 1024),
		"go_heap_objects":   ms.HeapObjects,
		"go_num_gc":         ms.NumGC,
		"go_stack_inuse_mb": ms.StackInuse / (1024 * 1024),
		"hub_channels":      len(h.channels),
		"hub_agents":        len(h.agents),
		"hub_threads":       len(h.threads),
		"hub_removed":       len(h.removedAgents),
		"hub_total_msgs":    totalMsgs,
		"hub_content_bytes": totalContent,
		"hub_collabs":       collabN,
		"hub_by_channel":    channels,
	}
}
