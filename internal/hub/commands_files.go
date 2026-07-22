package hub

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/pathutil"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// AddPendingReview adds a pending review to track
func (ch *CommandHandler) AddPendingReview(repoPath string, originalMsg *protocol.Message, agentName string) {
	ch.pendingMutex.Lock()
	defer ch.pendingMutex.Unlock()

	repoPath = normalizePendingReviewPath(repoPath)
	ch.pendingReviews[repoPath] = &protocol.PendingReview{
		OriginalMessage: originalMsg,
		RepoPath:        repoPath,
		RepoAgentName:   agentName,
		CreatedAt:       time.Now(),
	}
}

// GetPendingReview retrieves a pending review by repo path

// GetPendingReview retrieves a pending review by repo path
func (ch *CommandHandler) GetPendingReview(repoPath string) *protocol.PendingReview {
	ch.pendingMutex.Lock()
	defer ch.pendingMutex.Unlock()

	repoPath = normalizePendingReviewPath(repoPath)
	if pr := ch.pendingReviews[repoPath]; pr != nil {
		return pr
	}
	for k, pr := range ch.pendingReviews {
		if normalizePendingReviewPath(k) == repoPath {
			return pr
		}
	}
	return nil
}

// RemovePendingReview removes a pending review

// RemovePendingReview removes a pending review
func (ch *CommandHandler) RemovePendingReview(repoPath string) {
	ch.pendingMutex.Lock()
	defer ch.pendingMutex.Unlock()

	repoPath = normalizePendingReviewPath(repoPath)
	delete(ch.pendingReviews, repoPath)
}

// HasPendingReview checks if there's already a pending review for a path

// HasPendingReview checks if there's already a pending review for a path
func (ch *CommandHandler) HasPendingReview(repoPath string) bool {
	ch.pendingMutex.Lock()
	defer ch.pendingMutex.Unlock()

	repoPath = normalizePendingReviewPath(repoPath)
	if _, exists := ch.pendingReviews[repoPath]; exists {
		return true
	}
	for k := range ch.pendingReviews {
		if normalizePendingReviewPath(k) == repoPath {
			return true
		}
	}
	return false
}

func normalizePendingReviewPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(p)
}

// handleOpenFile validates and resolves a file path for opening in the editor.

// handleOpenFile validates and resolves a file path for opening in the editor.
func (ch *CommandHandler) handleOpenFile(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /open-file <file-path>"), nil
	}

	filePath := strings.Join(parts[1:], " ")
	wm := ch.hub.GetWorkspaceManager()
	if wm == nil {
		return ch.systemResponse(msg.Channel, "❌ Workspace manager is not available"), nil
	}
	var absPath string
	found := false
	for _, ws := range wm.ListWorkspaces() {
		candidate := filePath
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(ws.Path, candidate)
		}
		p, err := pathutil.WithinRoot(ws.Path, candidate)
		if err != nil {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			absPath = p
			found = true
			break
		}
	}
	if !found {
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("❌ File not found or path not under any workspace: %s", filePath)), nil
	}

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("📂 Opening file: %s", absPath)), nil
}

// handleAddWorkspace adds a new workspace

// handleAddWorkspace adds a new workspace
func (ch *CommandHandler) handleAddWorkspace(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /add-workspace <path> [name]"), nil
	}

	path := parts[1]
	name := path
	if len(parts) > 2 {
		name = strings.Join(parts[2:], " ")
	}

	wm := ch.hub.GetWorkspaceManager()
	if wm == nil {
		return ch.systemResponse(msg.Channel, "❌ Workspace manager is not available"), nil
	}
	ws, err := wm.AddWorkspace(name, path, AddWorkspaceOptions{Create: false})
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to add workspace: %v", err)), nil
	}
	if _, err := ch.EnsureHiddenRepoAgent(ctx, ws.Path, EnsureHiddenRepoAgentOptions{}); err != nil {
		log.Printf("[hidden-repo] /add-workspace ensure: %v", err)
	}

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("📁 Added workspace: %s\nPath: %s", ws.Name, ws.Path)), nil
}

// handleListWorkspaces lists all workspaces

// handleListWorkspaces lists all workspaces
func (ch *CommandHandler) handleListWorkspaces(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	wm := ch.hub.GetWorkspaceManager()
	if wm == nil {
		return ch.systemResponse(msg.Channel, "❌ Workspace manager is not available"), nil
	}
	workspaces := wm.ListWorkspaces()
	if len(workspaces) == 0 {
		return ch.systemResponse(msg.Channel, "📁 No workspaces configured"), nil
	}
	sort.Slice(workspaces, func(i, j int) bool {
		return strings.ToLower(workspaces[i].Name) < strings.ToLower(workspaces[j].Name)
	})

	var b strings.Builder
	b.WriteString("📁 Workspaces:\n")
	for _, ws := range workspaces {
		b.WriteString(fmt.Sprintf("• %s (%s)\n", ws.Name, ws.Path))
	}
	return ch.systemResponse(msg.Channel, strings.TrimSpace(b.String())), nil
}

// Assistant command handlers

// handleReminder handles reminder-related commands

func (ch *CommandHandler) handleGenerateImage(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel,
			"❌ Usage: `/generate-image <prompt>`\n\nExample: `/generate-image a minimal app icon for a biotech lab`\n\n"+
				"Uses local Ollama by default (`ollama pull x/flux2-klein:4b`). The image model is unloaded after each run (set `OLLAMA_IMAGE_KEEP_ALIVE=-1` to keep it loaded). Optional: `OLLAMA_IMAGE_MODEL`, `OLLAMA_ENDPOINT`, "+
				"`NEURAL_JUNKIE_IMAGE_PROVIDER=openai` + `OPENAI_API_KEY` for cloud."), nil
	}
	prompt := strings.TrimSpace(strings.Join(parts[1:], " "))
	if !ImageGenerationAvailable() {
		return ch.systemResponse(msg.Channel,
			"❌ Image generation is not configured. Pull an Ollama image model (e.g. `ollama pull x/flux2-klein:4b`) or set `NEURAL_JUNKIE_IMAGE_PROVIDER=none` to disable."), nil
	}
	progress := ch.systemResponse(msg.Channel, "🎨 Generating image…")
	if err := ch.hub.SendMessage(progress); err != nil {
		log.Printf("generate-image: failed to post progress message: %v", err)
	}
	from := ch.hub.resolveImagePostAgent(msg.Channel)
	if err := ch.hub.GenerateAndPostImage(ctx, msg.Channel, from, prompt, ""); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Image generation failed: %v", err)), nil
	}
	return ch.systemResponse(msg.Channel, "✅ Image generated and posted to the channel."), nil
}

func (ch *CommandHandler) handleGenerateMusic(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel,
			"❌ Usage: `/generate-music <style tags> [| lyrics]`\n\n"+
				"Example: `/generate-music lo-fi chill hip hop, 90 bpm | [Verse]\\nLate night code...`\n\n"+
				"Requires the **Music creation** pack and ACE-Step sidecar. Style tags are required; lyrics after `|` are optional (use `[Instrumental]` for no vocals)."), nil
	}
	raw := strings.TrimSpace(strings.Join(parts[1:], " "))
	style := raw
	lyrics := ""
	if i := strings.Index(raw, "|"); i >= 0 {
		style = strings.TrimSpace(raw[:i])
		lyrics = strings.TrimSpace(raw[i+1:])
	}
	if style == "" {
		return ch.systemResponse(msg.Channel, "❌ Style tags are required."), nil
	}
	if !ch.hub.MusicGenerationAvailable() {
		return ch.systemResponse(msg.Channel,
			"❌ Music generation is not available. Install and enable the **Music creation** pack, then run the ACE-Step setup script."), nil
	}
	progress := ch.systemResponse(msg.Channel, "🎵 Generating song…")
	if err := ch.hub.SendMessage(progress); err != nil {
		log.Printf("generate-music: failed to post progress message: %v", err)
	}
	from := ch.hub.resolveMusicPostAgent(msg.Channel)
	req := agent.MusicGenerateRequest{StyleTags: style, Lyrics: lyrics, DurationSec: 30}
	if lyrics == "" {
		req.Instrumental = true
		req.Lyrics = "[Instrumental]"
	} else if strings.EqualFold(lyrics, "[instrumental]") {
		req.Instrumental = true
		req.Lyrics = "[Instrumental]"
	}
	if err := ch.hub.GenerateAndPostMusic(ctx, msg.Channel, from, req); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Music generation failed: %v", err)), nil
	}
	return ch.systemResponse(msg.Channel, "✅ Song generated and posted to the channel."), nil
}

// handleAnalyzeDesign handles design analysis requests
func (ch *CommandHandler) handleAnalyzeDesign(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	imgs := protocol.ExtractUserImages(msg)
	if len(imgs) == 0 {
		return ch.systemResponse(msg.Channel,
			"❌ No image found for design analysis.\n\n"+
				"**Usage:**\n"+
				"1. Upload an image using the file picker\n"+
				"2. Type your message with @mentions to target specific agents\n"+
				"3. Send `/analyze-design` command\n\n"+
				"**Supported formats:** PNG, JPEG, WebP, GIF\n"+
				"**Max size:** 5MB per image"), nil
	}
	total := 0
	for _, p := range imgs {
		total += len(p.Data)
	}
	if total > 5*1024*1024 {
		return ch.systemResponse(msg.Channel,
			"❌ Image payload too large. Maximum total size is 5MB. Please compress images and try again."), nil
	}

	// Get channel agents
	channelAgents, err := ch.hub.GetChannelAgents(msg.Channel)
	if err != nil {
		return ch.systemResponse(msg.Channel, "❌ Failed to get channel agents"), nil
	}

	// Parse mentions from the message content
	mentionedAgentNames := protocol.ParseMentions(msg.Content)
	if len(mentionedAgentNames) == 0 {
		return ch.systemResponse(msg.Channel,
			"❌ No agents mentioned for design analysis.\n\n"+
				"**Usage:**\n"+
				"1. Upload an image\n"+
				"2. Type your message with @mentions (e.g., \"@FrontendAgent please analyze this design\")\n"+
				"3. Send `/analyze-design` command\n\n"+
				"**Available vision-capable agents:**\n"+
				ch.getVisionCapableAgentsList(channelAgents)), nil
	}

	// Find mentioned agents that support vision
	var targetAgents []protocol.AgentInfo
	for _, agent := range channelAgents {
		for _, mentionedName := range mentionedAgentNames {
			if strings.EqualFold(agent.Name, mentionedName) && agent.SupportsVision {
				targetAgents = append(targetAgents, agent)
				break
			}
		}
	}

	if len(targetAgents) == 0 {
		return ch.systemResponse(msg.Channel,
			"❌ No vision-capable agents found among mentioned agents.\n\n"+
				"**Mentioned agents:** "+strings.Join(mentionedAgentNames, ", ")+"\n"+
				"**Available vision-capable agents:**\n"+
				ch.getVisionCapableAgentsList(channelAgents)), nil
	}

	userImgMeta := make([]interface{}, 0, len(imgs))
	for _, p := range imgs {
		userImgMeta = append(userImgMeta, map[string]interface{}{
			"mime": p.MIME,
			"data": base64.StdEncoding.EncodeToString(p.Data),
		})
	}

	bodyContent := strings.TrimSpace(strings.Join(parts[1:], " "))
	if bodyContent == "" {
		bodyContent = strings.TrimSpace(msg.Content)
	}
	mentionStrip := regexp.MustCompile(`@\w+`)
	bodyWithoutMentions := strings.TrimSpace(mentionStrip.ReplaceAllString(bodyContent, ""))

	// Create analysis message for each target agent (single-agent mentions avoid duplicate fan-out)
	var agentNames []string
	for _, agent := range targetAgents {
		agentNames = append(agentNames, agent.Name)

		designMsg := &protocol.Message{
			ID:        protocol.NewMessage(protocol.MessageTypeChat, msg.Channel, msg.From, "").ID,
			Type:      protocol.MessageTypeChat,
			Channel:   msg.Channel,
			From:      msg.From,
			Content:   fmt.Sprintf("@%s %s", agent.Name, bodyWithoutMentions),
			Timestamp: msg.Timestamp,
			Metadata: map[string]interface{}{
				"design_analysis":           true,
				protocol.MetadataUserImages: userImgMeta,
			},
		}

		if err := ch.hub.SendMessage(designMsg); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to send design analysis request to %s: %v", agent.Name, err)), nil
		}
	}

	return ch.systemResponse(msg.Channel,
		"🎨 Design analysis started!\n\n"+
			"**Processing:** Analyzing design mockup...\n"+
			"**Target Agents:** "+strings.Join(agentNames, ", ")+"\n"+
			"**Output:** CSS style guide + HTML demo\n\n"+
			"The mentioned agents will analyze the design and generate:\n"+
			"• Complete CSS file with extracted styles\n"+
			"• HTML demo showcasing the design\n"+
			"• Markdown style guide with design tokens\n\n"+
			"Please wait for the analysis to complete..."), nil
}

// getVisionCapableAgentsList returns a formatted list of vision-capable agents

// getVisionCapableAgentsList returns a formatted list of vision-capable agents
func (ch *CommandHandler) getVisionCapableAgentsList(agents []protocol.AgentInfo) string {
	var visionAgents []string
	for _, agent := range agents {
		if agent.SupportsVision {
			visionAgents = append(visionAgents, "• @"+agent.Name)
		}
	}

	if len(visionAgents) == 0 {
		return "No vision-capable agents available in this channel."
	}

	return strings.Join(visionAgents, "\n")
}

// handleApproveFile approves a file change

// handleApproveFile approves a file change
func (ch *CommandHandler) handleApproveFile(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /approve-file <change-id>"), nil
	}

	changeID := parts[1]
	fileChangeManager := ch.hub.GetFileChangeManager()

	// Approve the file change
	change, err := fileChangeManager.ApproveFileChange(changeID, msg.From.ID)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to approve file change: %v", err)), nil
	}

	ch.hub.NotifyFileChangeApproved(change, msg.From.ID)

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("✅ File change approved and executed!\n\n"+
			"**Change ID:** %s\n"+
			"**Operation:** %s\n"+
			"**File:** %s\n"+
			"**Agent:** %s",
			change.ID, change.Operation, change.GetDisplayPath(), change.Agent.Name)), nil
}

// handleRejectFile rejects a file change

// handleRejectFile rejects a file change
func (ch *CommandHandler) handleRejectFile(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /reject-file <change-id> [reason]"), nil
	}

	changeID := parts[1]
	reason := "No reason provided"
	if len(parts) > 2 {
		reason = strings.Join(parts[2:], " ")
	}

	fileChangeManager := ch.hub.GetFileChangeManager()

	// Reject the file change
	change, err := fileChangeManager.RejectFileChange(changeID, msg.From.ID, reason)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to reject file change: %v", err)), nil
	}
	ch.hub.NotifyFileChangeRejected(change)

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("❌ File change rejected!\n\n"+
			"**Change ID:** %s\n"+
			"**Operation:** %s\n"+
			"**File:** %s\n"+
			"**Agent:** %s\n"+
			"**Reason:** %s",
			change.ID, change.Operation, change.GetDisplayPath(), change.Agent.Name, reason)), nil
}

// handleApproveDelete approves a delete operation (requires explicit confirmation)

// handleApproveDelete approves a delete operation (requires explicit confirmation)
func (ch *CommandHandler) handleApproveDelete(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /approve-delete <change-id>"), nil
	}

	changeID := parts[1]
	fileChangeManager := ch.hub.GetFileChangeManager()

	// Get the change first to verify it's a delete operation
	change, err := fileChangeManager.GetFileChange(changeID)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to get file change: %v", err)), nil
	}

	if !change.IsDeleteOperation() {
		return ch.systemResponse(msg.Channel, "❌ This is not a delete operation. Use /approve-file instead."), nil
	}

	// Approve the delete operation
	change, err = fileChangeManager.ApproveFileChange(changeID, msg.From.ID)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to approve delete operation: %v", err)), nil
	}

	ch.hub.NotifyFileChangeApproved(change, msg.From.ID)

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("🗑️ Delete operation approved and executed!\n\n"+
			"**Change ID:** %s\n"+
			"**File:** %s\n"+
			"**Agent:** %s\n\n"+
			"⚠️ File has been deleted and backed up.",
			change.ID, change.FilePath, change.Agent.Name)), nil
}

// handleListFileChanges lists all pending file changes

// handleListFileChanges lists all pending file changes
func (ch *CommandHandler) handleListFileChanges(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	fileChangeManager := ch.hub.GetFileChangeManager()
	pendingChanges := fileChangeManager.ListPendingFileChanges(msg.From.ID)

	if len(pendingChanges) == 0 {
		return ch.systemResponse(msg.Channel, "📝 No pending file changes found."), nil
	}

	var response strings.Builder
	response.WriteString("📝 **Pending File Changes:**\n\n")

	for i, change := range pendingChanges {
		timeRemaining := change.GetTimeRemaining()
		timeStr := "expired"
		if timeRemaining > 0 {
			timeStr = fmt.Sprintf("%.0f minutes", timeRemaining.Minutes())
		}

		response.WriteString(fmt.Sprintf("**%d.** `%s`\n", i+1, change.ID))
		response.WriteString(fmt.Sprintf("   • **Operation:** %s\n", change.Operation))
		response.WriteString(fmt.Sprintf("   • **File:** %s\n", change.GetDisplayPath()))
		response.WriteString(fmt.Sprintf("   • **Agent:** %s\n", change.Agent.Name))
		response.WriteString(fmt.Sprintf("   • **Time remaining:** %s\n", timeStr))

		if change.IsDeleteOperation() {
			response.WriteString(fmt.Sprintf("   • **⚠️ DELETE OPERATION** - Use `/approve-delete %s`\n", change.ID))
		} else {
			response.WriteString(fmt.Sprintf("   • **Approve:** `/approve-file %s`\n", change.ID))
		}
		response.WriteString(fmt.Sprintf("   • **Reject:** `/reject-file %s [reason]`\n\n", change.ID))
	}

	response.WriteString("💡 **Commands:**\n")
	response.WriteString("• `/approve-file <id>` - Approve a file change\n")
	response.WriteString("• `/reject-file <id> [reason]` - Reject a file change\n")
	response.WriteString("• `/approve-delete <id>` - Approve a delete operation\n")

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// handleIngestMeetings manually triggers Google meet notes sync
