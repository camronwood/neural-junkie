# test-everything — 2026-06-16-1955 UTC

- Hub: `http://127.0.0.1:18765`
- Full collab sweep (`FULL=1`): `True`
- Skip live: `False`
- Overall: **FAIL** (5/7 stages)

## Stage summary

| Stage | Status | Duration |
|-------|--------|----------|
| `test-all` | FAIL | 82s |
| `test-conversation-contract` | OK | 7s |
| `test-collab-plan` | OK | 1s |
| `test-scenario-assert` | OK | 0s |
| `collab-smoke` | OK | 1s |
| `learning-lora-smoke` | OK | 3s |
| `hub-health` | FAIL (start: make server-regression && make agents) | — |

## Artifacts

- Full log: `/Users/camronwood/development/projects/neural-junkie/docs/testing/test-everything-2026-06-16-1955.log`

## Failures (tail)

### test-all (exit 2)

```text
🔍 go vet...

🧪 Go tests...
?   	github.com/camronwood/neural-junkie	[no test files]
?   	github.com/camronwood/neural-junkie/cmd/agent	[no test files]
?   	github.com/camronwood/neural-junkie/cmd/chat	[no test files]
?   	github.com/camronwood/neural-junkie/cmd/cli	[no test files]
ok  	github.com/camronwood/neural-junkie/cmd/server	2.419s
?   	github.com/camronwood/neural-junkie/cmd/slack-oauth-relay	[no test files]
?   	github.com/camronwood/neural-junkie/cmd/tool-approval-hook	[no test files]
?   	github.com/camronwood/neural-junkie/cmd/verify-bootstrap-lora	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/agent	1.320s
ok  	github.com/camronwood/neural-junkie/internal/ai	0.315s
ok  	github.com/camronwood/neural-junkie/internal/cad	0.217s
ok  	github.com/camronwood/neural-junkie/internal/chatcontext	0.227s
ok  	github.com/camronwood/neural-junkie/internal/cli	3.756s
ok  	github.com/camronwood/neural-junkie/internal/codeindex	0.251s
ok  	github.com/camronwood/neural-junkie/internal/collaboration	0.262s
ok  	github.com/camronwood/neural-junkie/internal/collaboration/actions	0.240s
ok  	github.com/camronwood/neural-junkie/internal/collaboration/routing	0.232s
ok  	github.com/camronwood/neural-junkie/internal/collabworktree	0.301s
ok  	github.com/camronwood/neural-junkie/internal/config	3.140s
ok  	github.com/camronwood/neural-junkie/internal/confluence	0.259s
ok  	github.com/camronwood/neural-junkie/internal/delegation	0.229s
ok  	github.com/camronwood/neural-junkie/internal/embed	0.232s
ok  	github.com/camronwood/neural-junkie/internal/filechange	0.253s
ok  	github.com/camronwood/neural-junkie/internal/git	0.355s
ok  	github.com/camronwood/neural-junkie/internal/google/meetnotes	0.285s
ok  	github.com/camronwood/neural-junkie/internal/hardware	0.247s
ok  	github.com/camronwood/neural-junkie/internal/hfhub	4.028s
ok  	github.com/camronwood/neural-junkie/internal/hub	6.881s
ok  	github.com/camronwood/neural-junkie/internal/implementation/routing	0.242s
ok  	github.com/camronwood/neural-junkie/internal/integrations/slack	0.348s
ok  	github.com/camronwood/neural-junkie/internal/integrations/slack/relay	0.315s
ok  	github.com/camronwood/neural-junkie/internal/integrations/websearch	0.278s
ok  	github.com/camronwood/neural-junkie/internal/learning	0.267s
ok  	github.com/camronwood/neural-junkie/internal/lora/export	0.232s
ok  	github.com/camronwood/neural-junkie/internal/lora/train	0.231s
ok  	github.com/camronwood/neural-junkie/internal/lsp	0.200s
ok  	github.com/camronwood/neural-junkie/internal/mcp	0.655s
?   	github.com/camronwood/neural-junkie/internal/mcp/architecture	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/mcp/assistant	0.256s
?   	github.com/camronwood/neural-junkie/internal/mcp/backend	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/mcp/biology	0.251s
?   	github.com/camronwood/neural-junkie/internal/mcp/cad	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/mcp/codereview	0.231s
?   	github.com/camronwood/neural-junkie/internal/mcp/confluencemcp	[no test files]
?   	github.com/camronwood/neural-junkie/internal/mcp/database	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/mcp/devops	0.223s
ok  	github.com/camronwood/neural-junkie/internal/mcp/frontend	0.843s
?   	github.com/camronwood/neural-junkie/internal/mcp/repomcp	[no test files]
?   	github.com/camronwood/neural-junkie/internal/mcp/resources	[no test files]
?   	github.com/camronwood/neural-junkie/internal/mcp/rust	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/mcp/security	0.665s
ok  	github.com/camronwood/neural-junkie/internal/mcp/shared	0.239s
ok  	github.com/camronwood/neural-junkie/internal/mcp/web	0.247s
ok  	github.com/camronwood/neural-junkie/internal/mcp/workspace	0.238s
ok  	github.com/camronwood/neural-junkie/internal/mcp_export	0.197s
ok  	github.com/camronwood/neural-junkie/internal/memory	0.289s
ok  	github.com/camronwood/neural-junkie/internal/ollama	0.270s
ok  	github.com/camronwood/neural-junkie/internal/packs	0.282s
ok  	github.com/camronwood/neural-junkie/internal/pathutil	0.213s
ok  	github.com/camronwood/neural-junkie/internal/phoeniximport	0.236s
ok  	github.com/camronwood/neural-junkie/internal/protocol	0.238s
ok  	github.com/camronwood/neural-junkie/internal/repo	0.220s
ok  	github.com/camronwood/neural-junkie/internal/routing	0.256s
ok  	github.com/camronwood/neural-junkie/internal/scananalysis	0.376s
ok  	github.com/camronwood/neural-junkie/internal/scansummary	0.225s
?   	github.com/camronwood/neural-junkie/internal/secondaryanalysis	[no test files]
?   	github.com/camronwood/neural-junkie/internal/store/sqlite	[no test files]
?   	github.com/camronwood/neural-junkie/internal/testutil	[no test files]
?   	github.com/camronwood/neural-junkie/internal/workspacefiles	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/workspacesymbols	0.206s
?   	github.com/camronwood/neural-junkie/scenarios/fixtures/minimal-repo/collabs/c4dc37b6-813d-4817-b9af-a3e4255494e4/core/sample	[no test files]
?   	github.com/camronwood/neural-junkie/scenarios/fixtures/minimal-repo/collabs/eedc0e58-4926-4d26-8b02-66b7eeea25dd/core/sample	[no test files]
?   	github.com/camronwood/neural-junkie/scenarios/fixtures/minimal-repo/core/obscure/internal	[no test files]
?   	github.com/camronwood/neural-junkie/scenarios/fixtures/minimal-repo/core/sample	[no test files]
?   	github.com/camronwood/neural-junkie/scripts	[no test files]
ok  	github.com/camronwood/neural-junkie/test	31.093s

🧪 Desktop typecheck (tsc)...
src/components/RunbookBuilderPanel.tsx(163,11): error TS2322: Type 'void' is not assignable to type 'Collaboration'.
npm notice
npm notice New minor version of npm available! 11.11.0 -> 11.17.0
npm notice Changelog: https://github.com/npm/cli/releases/tag/v11.17.0
npm notice To update run: npm install -g npm@11.17.0
npm notice
make[1]: *** [test-all] Error 2
```

### hub-health (exit 1)

```text
(no output)
```

