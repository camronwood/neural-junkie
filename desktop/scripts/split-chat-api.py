#!/usr/bin/env python3
"""Split chatAPI.ts into modules (move-only refactor)."""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1] / "src" / "api"
CHATAPI = (ROOT / "chatAPI.ts").read_text()

class_start = CHATAPI.index("export class ChatAPI")
types_section = CHATAPI[:class_start].rstrip() + "\n"

rest = CHATAPI[class_start:]
body_start = rest.index("{") + 1
depth = 1
i = body_start
while i < len(rest) and depth > 0:
    if rest[i] == "{":
        depth += 1
    elif rest[i] == "}":
        depth -= 1
    i += 1
class_body = rest[body_start : i - 1]

methods: dict[str, str] = {}
privates: dict[str, str] = {}
order: list[str] = []

# Only top-level class members (exactly two leading spaces).
member_re = re.compile(
    r"^  ((?:/\*\*[\s\S]*?\*/\s*)?(?:private |public )?(?:async )?[a-z][\w]*\s*\([\s\S]*?\)(?:\s*:\s*[^\{]+)?)\s*\{",
    re.M,
)
for sig in member_re.finditer(class_body):
    start = sig.start()
    brace = sig.end() - 1
    depth = 1
    j = brace + 1
    while j < len(class_body) and depth > 0:
        if class_body[j] == "{":
            depth += 1
        elif class_body[j] == "}":
            depth -= 1
        j += 1
    chunk = class_body[start:j]
    sig_text = re.sub(r"/\*\*[\s\S]*?\*/", "", sig.group(1)).strip()
    name_m = re.search(r"(?:private |public )?(?:async )?([a-z][\w]*)", sig_text)
    name = name_m.group(1) if name_m else f"_anon_{len(order)}"
    is_private = "private " in sig_text.split("(")[0]
    if is_private:
        privates[name] = chunk
    else:
        if name in methods:
            name = f"{name}_{len(order)}"
        methods[name] = chunk
        order.append(name)

MODULE_MAP: dict[str, set[str]] = {
    "channels": {
        "createSession",
        "fetchMessages",
        "sendMessage",
        "sendMessageWithCredentials",
        "fetchChannels",
        "createChannel",
        "deleteChannel",
        "clearChannelHistory",
        "exportChannelHistory",
        "getChannelDurable",
        "setChannelDurable",
        "addAgentsToChannel",
        "removeAgentFromChannel",
        "fetchCommands",
        "clearCommandsCache",
        "channelInterject",
        "getWebSocketURL",
        "getThreadWebSocketURL",
        "fetchThreadMessages",
        "sendThreadReply",
        "fetchThreadMetadata",
        "testConnection",
        "fetchAssistantState",
        "markAssistantTaskDone",
        "dismissAssistantReminder",
        "getGoogleMeetNotesAppConfig",
        "saveGoogleMeetNotesAppConfig",
        "getGoogleMeetNotesStatus",
        "getGoogleMeetNotesAuthURL",
        "disconnectGoogleMeetNotes",
        "syncGoogleMeetNotes",
        "getSlackConfig",
        "saveSlackConfig",
        "getSlackStatus",
        "getSlackConnection",
        "getSlackBindings",
        "getSlackChannels",
        "saveSlackBinding",
        "deleteSlackBinding",
        "getSlackOAuthURL",
        "getSlackUserDMOAuthURL",
        "disconnectSlack",
        "restartSlackBridge",
        "getSlackInbox",
        "saveSlackInbox",
        "setSlackInboxAwayEnabled",
        "setSlackInboxForwardEnabled",
        "testSlackInboxDM",
        "slackTestPost",
    },
    "collaborations": {
        "fetchCollaborations",
        "readHubDataAccess",
        "acknowledgeCollaborationWorkspace",
        "collabTaskComplete",
        "collabTaskSkip",
        "collabTaskRedispatch",
        "collabTaskReassign",
        "collabPause",
        "collabResume",
        "approveCollabParticipantRequest",
        "denyCollabParticipantRequest",
        "collabTaskPost",
        "collabParticipantRequestPost",
    },
    "runbooks": {
        "createRunbook",
        "updateRunbook",
        "getRunbook",
        "suggestRunbookAssignee",
        "parseRunbookPlan",
        "submitRunbook",
        "startRunbook",
        "listRunbookTemplates",
        "createRunbookFromTemplate",
    },
    "agents": {
        "fetchAgents",
        "fetchAgentTools",
        "fetchChannelTools",
        "createDMAgent",
        "fetchCliAgentTypes",
        "fetchMyAgents",
        "deleteCachedAgent",
        "fetchRemovedAgents",
        "removeAgent",
        "deleteAgent",
        "recallAgent",
        "exportAgent",
        "switchAgentProvider",
        "switchAllAgentProviders",
        "restartConfiguredAgents",
        "fetchPendingToolApprovals",
        "approveToolCall",
        "rejectToolCall",
        "setAgentApprovalMode",
        "setAgentCustomRulesMarkdown",
        "setUserRulesMarkdown",
        "getUserRulesMarkdown",
    },
    "ollama": {
        "testAnthropicConnection",
        "testGitHubConnection",
        "testConfluenceConnection",
        "testOllamaConnection",
        "fetchOllamaStatus",
        "fetchOllamaModels",
        "testLMStudioConnection",
        "fetchLMStudioStatus",
        "fetchLMStudioModels",
        "fetchHfCatalog",
        "fetchHfStatus",
        "fetchProviders",
    },
    "files": {
        "fetchWorkspaces",
        "addWorkspace",
        "removeWorkspace",
        "fetchFiles",
        "fetchFileContent",
        "fetchScanSummaryWellImage",
        "fetchWorkspaceImageDataUrl",
        "saveFileContent",
        "renderCAD",
        "fetchCADMesh",
        "fetchCADParams",
        "fetchCADVersions",
        "saveCADVersion",
        "restoreCADVersion",
        "testOpenSCAD",
        "createFile",
        "renameFile",
        "deleteFile",
        "getGitStatus",
        "getGitDiff",
        "getGitFileSides",
        "gitAdd",
        "gitReset",
        "commitChanges",
        "pushChanges",
        "pullChanges",
        "searchWorkspaceFiles",
        "searchWorkspaceSymbols",
        "devFastEdit",
        "getGoLSPDiagnostics",
        "getLSPDiagnostics",
        "devComplete",
        "devAgentTurn",
        "repoSemanticSearch",
        "repoIndexStatus",
        "proposeFileChangeFromMessage",
        "listPendingFileChanges",
        "approveFileChange",
        "rejectFileChange",
        "getFileDiff",
    },
}

for mod, names in MODULE_MAP.items():
    for pn, pc in privates.items():
        if pn in names:
            methods[pn] = pc
            if pn not in order:
                order.append(pn)


def transform_method(chunk: str) -> str:
    lines = [line[2:] if line.startswith("  ") else line for line in chunk.splitlines()]
    body = "\n".join(lines)
    body = body.replace("this.hubFetch", "this.hub.hubFetch")
    body = body.replace("this.baseURL", "this.hub.baseURL")
    body = body.replace("this.commandsCache", "this.hub.commandsCache")
    body = body.replace("this.parsePacksMutationResponse", "parsePacksMutationResponse")
    return body


HUB_CLIENT = """import { hubAuthHeaders, hubSessionHeaders, normalizeHubBaseURL, getHubBaseURL } from '../../config/hubUrl';
import type { CommandDefinition } from '../../types/protocol';

export class HubClient {
  readonly baseURL: string;
  commandsCache: CommandDefinition[] | null = null;

  constructor(serverAddr: string = getHubBaseURL()) {
    this.baseURL = normalizeHubBaseURL(serverAddr);
  }

  hubHeaders(extra?: Record<string, string>): Record<string, string> {
    return {
      'Content-Type': 'application/json',
      ...hubAuthHeaders(),
      ...hubSessionHeaders(),
      ...extra,
    };
  }

  hubFetch(path: string, init?: RequestInit): Promise<Response> {
    const extra = (init?.headers as Record<string, string> | undefined) ?? {};
    const url = path.startsWith('http') ? path : `${this.baseURL}${path}`;
    return fetch(url, {
      ...init,
      headers: { ...this.hubHeaders(), ...extra },
    });
  }
}
"""

TYPE_IMPORTS = """import type {
  Message, AgentInfo, Channel, ThreadMetadata, CachedAgentInfo, ConnectionTestResult,
  FileChange, FileChangeDiff, CommandDefinition, AssistantStateResponse,
  GoogleMeetNotesStatus, GoogleMeetNotesAppConfig, SlackConfigResponse, SlackConnectionResponse,
  SlackStatus, SlackBinding, SlackChannelInfo, SlackPolicy, SlackInboxConfig,
  Collaboration, CollaborationTask, AssignSuggestion, ExecutionPolicy, GraphLayout,
  RunbookTemplate, AgentToolCapabilities, ChannelToolsResponse,
} from '../../types/protocol';
"""

EXTRA_IMPORTS = {
    "channels": "import { getHubAccessToken, setHubSessionToken } from '../../config/hubUrl';\nimport type { SendMessageResponse } from './types';\n",
    "files": "import type { CadParam } from './types';\n",
    "collaborations": "",
    "runbooks": "",
    "agents": "",
    "ollama": "",
}

CLASS_NAMES = {
    "channels": "ChannelsAPI",
    "collaborations": "CollaborationsAPI",
    "runbooks": "RunbooksAPI",
    "agents": "AgentsAPI",
    "ollama": "OllamaAPI",
    "files": "FilesAPI",
}

out_dir = ROOT / "chatAPI"
out_dir.mkdir(exist_ok=True)
(out_dir / "hubClient.ts").write_text(HUB_CLIENT)

assigned: set[str] = set()
for names in MODULE_MAP.values():
    assigned.update(names)

for mod, names in MODULE_MAP.items():
    chunks = [transform_method(methods[n]) for n in order if n in names and n in methods]
    class_name = CLASS_NAMES[mod]
    content = (
        f"import {{ HubClient }} from './hubClient';\n"
        f"{TYPE_IMPORTS}{EXTRA_IMPORTS.get(mod, '')}\n"
        f"export class {class_name} {{\n"
        f"  constructor(private readonly hub: HubClient) {{}}\n\n"
        f"{chr(10).join(chunks)}\n"
        f"}}\n"
    )
    (out_dir / f"{mod}.ts").write_text(content)

remaining = [n for n in order if n not in assigned and n in methods and n != "constructor"]
core_chunks = [transform_method(methods[n]) for n in remaining]
if "parsePacksMutationResponse" in privates:
    core_chunks.append(transform_method(privates["parsePacksMutationResponse"]))

(out_dir / "types.ts").write_text(types_section)

core_content = f"""import {{ HubClient }} from './hubClient';
import type {{
  PackStatus, PacksAPIResponse, PackCatalogEntry, PackValidationReport,
  CustomerPackContextResponse, InstallPackLoRAsResponse, ExpertPresetOption,
  LoraExpertContext, LoraTrainingBase, LoraTrainJob, LoraTrainStartRequest,
  UserLearning, LearningStats, LearningCategory, LearningScope,
}} from './types';
function parsePacksMutationResponse(data: Record<string, unknown>): PacksAPIResponse {{
  return {{
    packs: (data.packs as PackStatus[]) ?? [],
    pack_id: data.pack_id as string | undefined,
    layout_owner: data.layout_owner as string | undefined,
    layout_profile: data.layout_profile as string | undefined,
    capabilities: (data.capabilities as string[]) ?? [],
  }};
}}

export class ChatAPICore {{
  constructor(private readonly hub: HubClient) {{}}

{chr(10).join(core_chunks)}
}}
"""
(out_dir / "core.ts").write_text(core_content)

all_public = [n for n in order if n in methods and n not in privates]
delegations = []
for name in all_public:
    if name == "constructor":
        continue
    sig_line = re.sub(r"/\*\*[\s\S]*?\*/", "", methods[name].split("{", 1)[0]).strip()
    param_m = re.search(r"\(([\s\S]*)\)", sig_line)
    params = param_m.group(1).strip() if param_m else ""
    param_names = []
    for part in re.split(r",", params):
        part = part.strip()
        if not part:
            continue
        if part.startswith("..."):
            param_names.append("..." + part[3:].split(":")[0].strip())
        else:
            param_names.append(part.split(":")[0].split("=")[0].strip())
    call_args = ", ".join(param_names)
    mod = next((m for m, names in MODULE_MAP.items() if name in names), None)
    inst = {
        "channels": "channels",
        "collaborations": "collaborations",
        "runbooks": "runbooks",
        "agents": "agents",
        "ollama": "ollama",
        "files": "files",
    }.get(mod or "", "core")
    delegations.append(f"  {sig_line} {{\n    return this.{inst}.{name}({call_args});\n  }}")

facade = f"""export * from './chatAPI/types';
import {{ HubClient }} from './chatAPI/hubClient';
import {{ ChannelsAPI }} from './chatAPI/channels';
import {{ CollaborationsAPI }} from './chatAPI/collaborations';
import {{ RunbooksAPI }} from './chatAPI/runbooks';
import {{ AgentsAPI }} from './chatAPI/agents';
import {{ OllamaAPI }} from './chatAPI/ollama';
import {{ FilesAPI }} from './chatAPI/files';
import {{ ChatAPICore }} from './chatAPI/core';

export class ChatAPI {{
  private readonly hub: HubClient;
  private readonly channels: ChannelsAPI;
  private readonly collaborations: CollaborationsAPI;
  private readonly runbooks: RunbooksAPI;
  private readonly agents: AgentsAPI;
  private readonly ollama: OllamaAPI;
  private readonly files: FilesAPI;
  private readonly core: ChatAPICore;

  constructor(serverAddr?: string) {{
    this.hub = new HubClient(serverAddr);
    this.channels = new ChannelsAPI(this.hub);
    this.collaborations = new CollaborationsAPI(this.hub);
    this.runbooks = new RunbooksAPI(this.hub);
    this.agents = new AgentsAPI(this.hub);
    this.ollama = new OllamaAPI(this.hub);
    this.files = new FilesAPI(this.hub);
    this.core = new ChatAPICore(this.hub);
  }}

{chr(10).join(delegations)}
}}
"""
(ROOT / "chatAPI.ts").write_text(facade)

print(f"Wrote modules to {out_dir}")
print(f"Public methods: {len(all_public)}")
print(f"Remaining in core: {remaining}")
print(f"Unmapped: {[n for n in all_public if n not in assigned and n not in remaining]}")
