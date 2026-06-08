package agent

import (
	"context"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// shouldDeferImplSessionForCombinedDeliveryExport reports write+save in one turn with
// no prior assistant body — chat generates the article first, then we propose a file.
func shouldDeferImplSessionForCombinedDeliveryExport(a *Agent, msg *protocol.Message) bool {
	if a == nil || msg == nil {
		return false
	}
	if !userRequestsContentDelivery(msg.Content) {
		return false
	}
	if !userRequestsFileExportForMessage(msg) {
		return false
	}
	history := a.historyForPriorReference(msg.Channel)
	return findPriorAssistantContent(history, msg.ID, a.Info.ID, priorReferenceMinChars) == ""
}

func defaultContentDeliveryExportPath(msg *protocol.Message) string {
	if msg == nil {
		return "article.md"
	}
	if p := preferFileExportTargetPath(msg); p != "" {
		return p
	}
	lower := strings.ToLower(msg.Content)
	if strings.Contains(lower, "linkedin") {
		return "linkedin-article.md"
	}
	return "article.md"
}

// extractContentDeliveryBodyForExport pulls markdown/article text from a chat reply.
func extractContentDeliveryBodyForExport(response string) string {
	trim := strings.TrimSpace(response)
	if trim == "" {
		return ""
	}
	lower := strings.ToLower(trim)
	for _, prefix := range []string{
		"certainly!",
		"sure,",
		"sure!",
		"here's a draft",
		"here is a draft",
		"here's your",
		"here is your",
	} {
		if strings.HasPrefix(lower, prefix) {
			if idx := strings.Index(trim, "\n"); idx >= 0 {
				trim = strings.TrimSpace(trim[idx+1:])
				lower = strings.ToLower(trim)
			}
			break
		}
	}
	if idx := strings.Index(trim, "\n---\n"); idx >= 0 {
		trim = strings.TrimSpace(trim[idx+len("\n---\n"):])
	} else if strings.HasPrefix(trim, "---\n") {
		trim = strings.TrimSpace(trim[4:])
	}
	return strings.TrimSpace(trim)
}

// maybeProposeCombinedDeliveryExport proposes a file after chat generated the article body.
func (a *Agent) maybeProposeCombinedDeliveryExport(ctx context.Context, msg *protocol.Message, response string) (string, bool, error) {
	if a == nil || msg == nil || !shouldDeferImplSessionForCombinedDeliveryExport(a, msg) {
		return response, false, nil
	}
	body := extractContentDeliveryBodyForExport(response)
	if len(body) < 200 {
		return response, false, nil
	}
	if looksLikePlaceholderProposalContent(body) {
		return response, false, nil
	}
	path := defaultContentDeliveryExportPath(msg)
	if err := a.proposeFileChangePreferEditOrCreate(ctx, msg.Channel, path, body, msg); err != nil {
		return response, false, err
	}
	log.Printf("[%s] combined_delivery_export(path=%s,len=%d)", a.Info.Name, path, len(body))
	cleaned := strings.TrimSpace(response)
	if cleaned == "" {
		cleaned = "I drafted the content and submitted a file change proposal for your approval."
	} else if !strings.Contains(strings.ToLower(cleaned), "file change proposal") {
		cleaned += "\n\nI submitted a file change proposal for `" + path + "` for your approval."
	}
	return cleaned, true, nil
}
