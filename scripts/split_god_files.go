//go:build ignore

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type assignment struct {
	file  string
	names []string
}

var agentAssign = []assignment{
	{file: "agent_lifecycle.go", names: []string{
		"Start", "StartMultiChannel", "AddChannel", "replayUnrespondedHistory",
		"processUnrespondedHistory", "discoverChannels", "discoverChannelsOnce", "Stop",
		"unrespondedHistoryReplayDelay",
	}},
	{file: "agent_message.go", names: []string{
		"handleMessage", "shouldRespond", "sendThinkingStatus", "effectiveChannelType",
		"getTypeKeywords", "allowCustomChannelBroadPromptReply", "looksLikeUserRequest",
		"shouldInjectWorkspaceCode", "shouldTreatCapabilityAsCodeRequest",
		"isCustomChannelPrompt", "isAgentType",
	}},
	{file: "agent_response.go", names: []string{
		"generateResponse", "generateResponseStreaming", "collectStreamTokens",
		"agentHasDedicatedContext", "truncationLabelForError", "appendFileChangeMachineBlockDocs",
		"collectIncludedFilePaths",
	}},
	{file: "agent_prompt.go", names: []string{
		"resolveWorkspacePath", "buildPrompt", "channelHistory", "channelHistorySafe",
		"replaceChannelHistory", "historyChannelNames", "addToHistory",
		"getAgentTypeInstructions", "getResponseLengthGuidanceForMessage", "getResponseLengthGuidance",
	}},
	{file: "agent_collab_turn.go", names: []string{
		"promptNextCollaborationTurn", "collabTaskRateLimitOK", "isHumanCollabSpeaker",
		"collabOutOfTurnMentionOK", "taskAssigneeFromMetadata", "recapAssigneeFromMetadata",
		"collaborationWorkingDirectoryForMessage",
	}},
}

var agentCore = map[string]bool{
	"NewAgent": true, "NewAgentWithProvider": true, "registerGenCancel": true,
	"unregisterGenCancel": true, "AbortChannel": true, "AbortAllChannels": true,
	"RegisterGenCancelForTest": true, "ActiveGenCountForTest": true,
	"Agent": true, "MCPServerInterface": true, "HubClient": true,
	"CollaborationClient": true, "CollaborationInfo": true,
	"CollaborationAgentSummary": true, "ExportableAgent": true, "ConversationContext": true,
	"customChannelBroadPromptResponderCap": true, "customChannelRelevanceMinScore": true,
	"collabTaskMinReplyInterval": true,
}

var hubAssign = []assignment{
	{file: "hub_subscribe.go", names: []string{
		"Subscribe", "Unsubscribe", "broadcast", "BroadcastDirect",
		"SubscribeToThread", "UnsubscribeFromThread", "broadcastToThread",
	}},
	{file: "hub_dispatch.go", names: []string{
		"SendMessage", "inheritCollaborationFromChannel", "shouldParseCollaborationMentions",
		"maybeRequestCollaborationParticipants", "sendParticipantAddRequestNotice",
		"ApproveCollaborationParticipantRequest", "DenyCollaborationParticipantRequest",
		"processCollaborationLifecycle", "maybeIngestPlanArtifact", "maybeUpdateTaskStatus",
		"finalizeAndBroadcastCollaboration", "normalizeTaskStatus", "attachCollaborationData",
		"shouldAutoCreateRepoAgent", "messageHasSharedWorkspaceForRepo", "sameRepoPath",
		"autoCreateRepoAgent", "persistCollaborationReviewAssets", "shortCollabID",
		"RedispatchOpenCollaborationTasksAfterSessionRestore", "dispatchReadyCollabTasks",
		"DispatchReadyCollabTasksForSnapshot", "dispatchCollabTaskMessages",
		"dispatchCollabTaskMessagesFilter", "formatCollabTaskDispatchBody",
		"NewCollaborationClientAdapter", "collabClientAdapter", "collaborationInfoForAgent",
		"registerFileChangeProposal", "resolveWorkspacePath", "resolveWorkspaceRoot",
	}},
	{file: "hub_agent_registry.go", names: []string{
		"RegisterAgent", "UnregisterAgent", "getAgentListString", "JoinChannel",
		"shouldSkipJoinAnnouncementLocked", "LeaveChannel", "GetChannelAgents", "GetAgent",
		"FindLiveAgentByDisplayName", "syncAgentInfoCopiesInChannelsLocked",
		"SyncAgentRegistration", "ListAgents", "GetRemovedAgents", "IsAgentRemoved",
		"AddRemovedAgent", "RemoveFromRemovedAgents", "IsAgentInAnyChannel",
		"ensureAgentSubscribed", "GetAgentChannels", "AddAgentToChannel", "RemoveAgentFromChannel",
	}},
}

var hubCore = map[string]bool{
	"Hub": true, "NewHub": true, "SessionSnapshot": true, "SessionSaveHealth": true,
	"ChannelSnapshot": true, "ThreadSnapshot": true, "MetadataKeyHistoryResync": true,
	"DefaultSessionPath": true,
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	if err := split(filepath.Join(root, "internal/agent/agent.go"), "agent.go", agentAssign, agentCore); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := split(filepath.Join(root, "internal/hub/hub.go"), "hub.go", hubAssign, hubCore); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func split(srcPath, defaultFile string, assigns []assignment, core map[string]bool) error {
	src, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	content := string(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, srcPath, content, parser.ParseComments)
	if err != nil {
		return err
	}

	target := map[string]string{}
	for _, a := range assigns {
		for _, n := range a.names {
			target[n] = a.file
		}
	}

	type piece struct {
		start int
		end   int
		file  string
	}

	var pieces []piece
	for i, decl := range f.Decls {
		start := fset.Position(decl.Pos()).Offset
		end := fset.Position(decl.End()).Offset
		if i+1 < len(f.Decls) {
			end = fset.Position(f.Decls[i+1].Pos()).Offset
		} else {
			end = len(content)
		}
		block := content[start:end]
		name := declName(decl)
		file := defaultFile
		if name != "" {
			if core[name] {
				file = defaultFile
			} else if t, ok := target[name]; ok {
				file = t
			} else if strings.Contains(block, "*collabClientAdapter)") {
				file = "hub_dispatch.go"
			}
		}
		pieces = append(pieces, piece{start: start, end: end, file: file})
	}

	header := content[:pieces[0].start]
	for i := 1; i < len(pieces); i++ {
		if pieces[i].start != pieces[i-1].end {
			return fmt.Errorf("gap/overlap between decls at %d in %s", pieces[i].start, srcPath)
		}
	}
	if pieces[len(pieces)-1].end != len(content) {
		return fmt.Errorf("tail bytes unconsumed in %s", srcPath)
	}

	buckets := map[string][]string{defaultFile: {}}
	for _, a := range assigns {
		buckets[a.file] = nil
	}
	for _, p := range pieces {
		buckets[p.file] = append(buckets[p.file], content[p.start:p.end])
	}

	dir := filepath.Dir(srcPath)
	if err := os.Remove(srcPath); err != nil {
		return err
	}
	for file, blocks := range buckets {
		if len(blocks) == 0 {
			return fmt.Errorf("empty bucket %s", file)
		}
		out := filepath.Join(dir, file)
		body := header + strings.Join(blocks, "")
		if err := os.WriteFile(out, []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func declName(d ast.Decl) string {
	switch x := d.(type) {
	case *ast.FuncDecl:
		if x.Name != nil {
			return x.Name.Name
		}
	case *ast.GenDecl:
		if len(x.Specs) == 0 {
			return ""
		}
		switch s := x.Specs[0].(type) {
		case *ast.TypeSpec:
			if s.Name != nil {
				return s.Name.Name
			}
		case *ast.ValueSpec:
			if len(s.Names) > 0 {
				return s.Names[0].Name
			}
		}
	}
	return ""
}
