package agent

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

const cliChatAttachmentsDir = ".nj-chat-attachments"

// cliWorkDirUsable reports whether workDir exists and is writable (placeholder env paths are ignored).
func cliWorkDirUsable(workDir string) bool {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		return false
	}
	abs, err := filepath.Abs(workDir)
	if err != nil {
		return false
	}
	fi, err := os.Stat(abs)
	if err != nil || !fi.IsDir() {
		return false
	}
	probe := filepath.Join(abs, ".nj-cli-workdir-probe")
	if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
		return false
	}
	_ = os.Remove(probe)
	return true
}

func (a *Agent) isCLIAgent() bool {
	return a.Info.Type == protocol.AgentTypeCLI || isCLIProvider(a.Info.AIProvider)
}

// resolveCLIWorkDir returns the subprocess working directory for CLI agents.
// Priority: explicit WorkDirEnv override, then message/collaboration workspace,
// then the provider default from hub startup, then process cwd.
func (a *Agent) resolveCLIWorkDir(msg *protocol.Message) string {
	provider := strings.ToLower(strings.TrimSpace(a.Info.AIProvider))
	for _, cfg := range cliAgentRegistry {
		if cfg.ProviderName == provider && cfg.WorkDirEnv != "" {
			if v := strings.TrimSpace(os.Getenv(cfg.WorkDirEnv)); v != "" && cliWorkDirUsable(v) {
				return v
			}
		}
	}
	if ws := a.resolveWorkspacePath(msg); ws != "" {
		return ws
	}
	if p, ok := a.GetAIProvider().(*ai.CLIAgentProvider); ok {
		if wd := strings.TrimSpace(p.WorkDir); wd != "" {
			return wd
		}
	}
	wd, _ := os.Getwd()
	return wd
}

// prepareCLIInvocation applies per-message workdir, approval mode, and optional judge model
// overrides before a CLI subprocess runs. Call the returned function after the invocation
// completes to restore any temporary model override.
func (a *Agent) prepareCLIInvocation(msg *protocol.Message) func() {
	if !a.isCLIAgent() {
		return func() {}
	}
	p, ok := a.GetAIProvider().(*ai.CLIAgentProvider)
	if !ok {
		return func() {}
	}
	if wd := a.resolveCLIWorkDir(msg); wd != "" {
		p.WorkDir = wd
	}
	mode := strings.TrimSpace(a.Info.ApprovalMode)
	if mode == "" {
		provider := strings.ToLower(strings.TrimSpace(a.Info.AIProvider))
		for _, cfg := range cliAgentRegistry {
			if cfg.ProviderName == provider && strings.TrimSpace(cfg.ApprovalMode) != "" {
				mode = cfg.ApprovalMode
				break
			}
		}
	}
	if mode != "" {
		p.ApprovalMode = mode
	}

	var restoreModel func()
	if isDeliverableJudgeMessage(msg) && p.ProviderName == "gemini-cli" {
		judgeModel := resolveDeliverableJudgeGeminiModel()
		if judgeModel != "" {
			prevModel := strings.TrimSpace(p.Env["GEMINI_MODEL"])
			if prevModel == "" {
				prevModel = p.EffectiveCLIModel()
			}
			p.Env["GEMINI_MODEL"] = judgeModel
			ai.SetCLIProviderModelOverride("gemini-cli", judgeModel)
			restoreModel = func() {
				if prevModel != "" {
					p.Env["GEMINI_MODEL"] = prevModel
					ai.SetCLIProviderModelOverride("gemini-cli", prevModel)
				} else {
					delete(p.Env, "GEMINI_MODEL")
					ai.SetCLIProviderModelOverride("gemini-cli", "")
				}
			}
		}
	}
	if restoreModel == nil {
		return func() {}
	}
	return restoreModel
}

func extensionForImageMIME(mime string) string {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/png":
		return ".png"
	default:
		return ".png"
	}
}

func safeCLIAttachmentSegment(s string) string {
	if s == "" {
		return "msg"
	}
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "msg"
	}
	if len(out) > 64 {
		return out[:64]
	}
	return out
}

// MaterializeUserImagesForCLI writes chat images under workDir/.nj-chat-attachments/{messageID}/.
// Returns paths relative to workDir for inclusion in the CLI prompt.
func MaterializeUserImagesForCLI(workDir, messageID string, imgs []protocol.UserImagePart) ([]string, error) {
	if len(imgs) == 0 {
		return nil, nil
	}
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		return nil, fmt.Errorf("empty CLI work directory")
	}
	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return nil, err
	}
	seg := safeCLIAttachmentSegment(messageID)
	if seg == "msg" {
		seg = uuid.New().String()
	}
	dir := filepath.Join(absWork, cliChatAttachmentsDir, seg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create attachment dir: %w", err)
	}

	var relPaths []string
	for i, im := range imgs {
		ext := extensionForImageMIME(im.MIME)
		name := fmt.Sprintf("img-%d%s", i+1, ext)
		absPath := filepath.Join(dir, name)
		if err := os.WriteFile(absPath, im.Data, 0o644); err != nil {
			return relPaths, fmt.Errorf("write %s: %w", name, err)
		}
		rel, err := filepath.Rel(absWork, absPath)
		if err != nil {
			rel = filepath.Join(cliChatAttachmentsDir, seg, name)
		}
		relPaths = append(relPaths, filepath.ToSlash(rel))
	}
	return relPaths, nil
}

func formatCLIAttachedImagesSection(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n=== ATTACHED IMAGES (on disk) ===\n")
	b.WriteString("The user attached image(s) saved under your working directory. Read these files to see the images:\n")
	for _, p := range paths {
		b.WriteString("- ")
		b.WriteString(p)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

func (a *Agent) augmentPromptWithCLIImages(msg *protocol.Message, prompt string) string {
	if msg == nil || !a.isCLIAgent() {
		return prompt
	}
	imgs := protocol.ExtractUserImages(msg)
	if len(imgs) == 0 {
		return prompt
	}
	workDir := a.resolveCLIWorkDir(msg)
	msgID := msg.ID
	if msgID == "" {
		msgID = uuid.New().String()
	}
	paths, err := MaterializeUserImagesForCLI(workDir, msgID, imgs)
	if err != nil {
		log.Printf("[%s] CLI image bridge: %v", a.Info.Name, err)
		return prompt + "\n(User attached image(s) but they could not be saved to disk for CLI access. Ask the user to save the image under your work directory and reference the path.)\n"
	}
	return prompt + formatCLIAttachedImagesSection(paths)
}
