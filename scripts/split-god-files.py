#!/usr/bin/env python3
"""Move-only split by top-level declaration boundaries."""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

AGENT_TARGETS = {
    "agent_lifecycle.go": {
        "Start", "StartMultiChannel", "AddChannel", "replayUnrespondedHistory",
        "processUnrespondedHistory", "discoverChannels", "discoverChannelsOnce", "Stop",
        "unrespondedHistoryReplayDelay",
    },
    "agent_message.go": {
        "handleMessage", "shouldRespond", "sendThinkingStatus", "effectiveChannelType",
        "getTypeKeywords", "allowCustomChannelBroadPromptReply", "looksLikeUserRequest",
        "shouldInjectWorkspaceCode", "shouldTreatCapabilityAsCodeRequest",
        "isCustomChannelPrompt", "isAgentType",
    },
    "agent_response.go": {
        "generateResponse", "generateResponseStreaming", "collectStreamTokens",
        "agentHasDedicatedContext", "truncationLabelForError", "appendFileChangeMachineBlockDocs",
        "collectIncludedFilePaths",
    },
    "agent_prompt.go": {
        "resolveWorkspacePath", "buildPrompt", "channelHistory", "channelHistorySafe",
        "replaceChannelHistory", "historyChannelNames", "addToHistory",
        "getAgentTypeInstructions", "getResponseLengthGuidanceForMessage", "getResponseLengthGuidance",
    },
    "agent_collab_turn.go": {
        "promptNextCollaborationTurn", "collabTaskRateLimitOK", "isHumanCollabSpeaker",
        "collabOutOfTurnMentionOK", "taskAssigneeFromMetadata", "recapAssigneeFromMetadata",
        "collaborationWorkingDirectoryForMessage",
    },
}

HUB_TARGETS = {
    "hub_subscribe.go": {
        "Subscribe", "Unsubscribe", "broadcast", "BroadcastDirect",
        "SubscribeToThread", "UnsubscribeFromThread", "broadcastToThread",
    },
    "hub_dispatch.go": {
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
    },
    "hub_agent_registry.go": {
        "RegisterAgent", "UnregisterAgent", "getAgentListString", "JoinChannel",
        "shouldSkipJoinAnnouncementLocked", "LeaveChannel", "GetChannelAgents", "GetAgent",
        "FindLiveAgentByDisplayName", "syncAgentInfoCopiesInChannelsLocked",
        "SyncAgentRegistration", "ListAgents", "GetRemovedAgents", "IsAgentRemoved",
        "AddRemovedAgent", "RemoveFromRemovedAgents", "IsAgentInAnyChannel",
        "ensureAgentSubscribed", "GetAgentChannels", "AddAgentToChannel", "RemoveAgentFromChannel",
    },
}

AGENT_CORE = {
    "NewAgent", "NewAgentWithProvider", "registerGenCancel", "unregisterGenCancel",
    "AbortChannel", "AbortAllChannels", "RegisterGenCancelForTest", "ActiveGenCountForTest",
    "Agent", "MCPServerInterface", "HubClient", "CollaborationClient", "CollaborationInfo",
    "CollaborationAgentSummary", "ExportableAgent", "ConversationContext",
    "customChannelBroadPromptResponderCap", "customChannelRelevanceMinScore", "collabTaskMinReplyInterval",
}

HUB_CORE = {
    "Hub", "NewHub", "SessionSnapshot", "SessionSaveHealth", "ChannelSnapshot",
    "ThreadSnapshot", "MetadataKeyHistoryResync", "DefaultSessionPath",
}

DECL = re.compile(r"^(func|type|const|var)\b")


def decl_name(first_decl_line: str) -> str:
    if first_decl_line.startswith("func "):
        m = re.match(r"func\s+(?:\([^)]+\)\s+)?(\w+)", first_decl_line)
        return m.group(1) if m else ""
    if first_decl_line.startswith("type "):
        m = re.match(r"type\s+(\w+)", first_decl_line)
        return m.group(1) if m else ""
    if first_decl_line.startswith("const "):
        m = re.match(r"const\s+(\w+)", first_decl_line)
        return m.group(1) if m else ""
    if first_decl_line.startswith("var "):
        m = re.match(r"var\s+(\w+)", first_decl_line)
        return m.group(1) if m else ""
    return ""


def end_of_decl(lines: list[str], decl_line: int) -> int:
    line = lines[decl_line]
    if line.startswith("const (") or line.startswith("var ("):
        i = decl_line
        while i < len(lines) and ")" not in lines[i]:
            i += 1
        return i

    brace = 0
    saw_brace = False
    i = decl_line
    while i < len(lines):
        for ch in lines[i]:
            if ch == "{":
                brace += 1
                saw_brace = True
            elif ch == "}":
                brace -= 1
        if saw_brace and brace == 0:
            return i
        if not saw_brace and i == decl_line and line.rstrip().endswith(";"):
            return i
        i += 1
    return len(lines) - 1


def iter_decls(lines: list[str]):
    i = 0
    n = len(lines)
    while i < n:
        while i < n and lines[i].strip() == "":
            i += 1
        if i >= n:
            break
        start = i
        while i < n:
            if lines[i].strip().startswith("//"):
                i += 1
                continue
            if lines[i].strip() == "":
                i += 1
                continue
            break
        if i >= n:
            break
        if not DECL.match(lines[i]) and not lines[i].startswith("const (") and not lines[i].startswith("var ("):
            raise RuntimeError(f"unexpected line {i+1}: {lines[i]!r}")
        name = decl_name(lines[i])
        end = end_of_decl(lines, i)
        block = "".join(lines[start : end + 1])
        yield name, block
        i = end + 1


def pick_file(name: str, block: str, targets: dict[str, set[str]], core: set[str], default: str) -> str:
    if "*collabClientAdapter)" in block:
        return "hub_dispatch.go"
    if name in core:
        return default
    for fname, names in targets.items():
        if name in names:
            return fname
    if not name:
        for fname, names in targets.items():
            for n in names:
                if re.search(rf"\b{n}\b", block):
                    return fname
        for n in core:
            if re.search(rf"\b{n}\b", block):
                return default
    return default


def split_one(src: Path, targets: dict[str, set[str]], core: set[str], default: str) -> list[Path]:
    raw = src.read_text()
    lines = raw.splitlines(keepends=True)

    pkg = ""
    imports = ""
    i = 0
    while i < len(lines):
        if lines[i].strip() == "":
            i += 1
            continue
        if lines[i].startswith("package "):
            pkg = lines[i]
            i += 1
            continue
        if lines[i].startswith("import "):
            imp_start = i
            if "(" in lines[i]:
                i += 1
                while i < len(lines) and ")" not in lines[i]:
                    i += 1
                i += 1
            else:
                i += 1
            imports = "".join(lines[imp_start:i])
            continue
        break
    body_lines = lines[i:]

    blocks_list: list[tuple[str, str]] = []
    for name, block in iter_decls(body_lines):
        blocks_list.append((name, block))

    consumed = sum(len(b) for _, b in blocks_list)
    remainder = "".join(body_lines)[consumed:]
    if remainder.strip():
        raise RuntimeError(f"{src.name}: unconsumed tail: {remainder[:120]!r}")
    if blocks_list and remainder:
        n, b = blocks_list[-1]
        blocks_list[-1] = (n, b + remainder)

    buckets: dict[str, list[str]] = {default: []}
    for t in targets:
        buckets.setdefault(t, [])

    for name, block in blocks_list:
        fname = pick_file(name, block, targets, core, default)
        buckets.setdefault(fname, []).append(block)

    written: list[Path] = []
    for fname, blocks in buckets.items():
        if not blocks:
            raise RuntimeError(f"{src.name}: empty bucket {fname}")
        out = src.parent / fname
        header = pkg + "\n"
        if fname != default:
            header += "\n" + imports + "\n"
        else:
            header += "\n" + imports + "\n"
        out.write_text(header + "".join(blocks))
        written.append(out)

    src.unlink()
    return written


def main() -> int:
    agent_written = split_one(
        ROOT / "internal/agent/agent.go", AGENT_TARGETS, AGENT_CORE, "agent.go"
    )
    hub_written = split_one(
        ROOT / "internal/hub/hub.go", HUB_TARGETS, HUB_CORE, "hub.go"
    )
    paths = agent_written + hub_written
    subprocess.run(
        ["go", "run", "golang.org/x/tools/cmd/goimports@latest", "-w", *[str(p) for p in paths]],
        check=True,
        cwd=ROOT,
    )
    subprocess.run(["go", "test", "./internal/agent/...", "./internal/hub/..."], check=True, cwd=ROOT)

    print("=== agent ===")
    for p in sorted((ROOT / "internal/agent").glob("agent*.go")):
        print(f"{p.name}: {len(p.read_text().splitlines())}")
    print("=== hub ===")
    for p in sorted((ROOT / "internal/hub").glob("hub*.go")):
        if p.name in {"hub.go", "hub_subscribe.go", "hub_dispatch.go", "hub_agent_registry.go"}:
            print(f"{p.name}: {len(p.read_text().splitlines())}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
