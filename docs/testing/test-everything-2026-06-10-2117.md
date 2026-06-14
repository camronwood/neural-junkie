# test-everything — 2026-06-10-2117 UTC

- Hub: `http://127.0.0.1:18765`
- Full collab sweep (`FULL=1`): `False`
- Skip live: `False`
- Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration |
|-------|--------|----------|
| `test-all` | FAIL | 75s |

## Artifacts

- Full log: `/Users/camronwood/development/projects/neural-junkie/docs/testing/test-everything-2026-06-10-2117.log`

## Failures (tail)

### test-all (exit 2)

```text
🔍 go vet...

🧪 Go tests...
?   	github.com/camronwood/neural-junkie	[no test files]
?   	github.com/camronwood/neural-junkie/cmd/agent	[no test files]
?   	github.com/camronwood/neural-junkie/cmd/chat	[no test files]
?   	github.com/camronwood/neural-junkie/cmd/cli	[no test files]
ok  	github.com/camronwood/neural-junkie/cmd/server	1.882s
?   	github.com/camronwood/neural-junkie/cmd/slack-oauth-relay	[no test files]
?   	github.com/camronwood/neural-junkie/cmd/tool-approval-hook	[no test files]
?   	github.com/camronwood/neural-junkie/cmd/verify-bootstrap-lora	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/agent	1.360s
ok  	github.com/camronwood/neural-junkie/internal/ai	0.276s
ok  	github.com/camronwood/neural-junkie/internal/cad	0.153s
ok  	github.com/camronwood/neural-junkie/internal/chatcontext	0.169s
ok  	github.com/camronwood/neural-junkie/internal/cli	3.412s
ok  	github.com/camronwood/neural-junkie/internal/codeindex	0.237s
ok  	github.com/camronwood/neural-junkie/internal/collaboration	0.243s
ok  	github.com/camronwood/neural-junkie/internal/collaboration/actions	0.247s
ok  	github.com/camronwood/neural-junkie/internal/collaboration/routing	0.277s
ok  	github.com/camronwood/neural-junkie/internal/collabworktree	0.306s
ok  	github.com/camronwood/neural-junkie/internal/config	3.816s
ok  	github.com/camronwood/neural-junkie/internal/confluence	0.279s
ok  	github.com/camronwood/neural-junkie/internal/delegation	0.213s
ok  	github.com/camronwood/neural-junkie/internal/embed	0.277s
ok  	github.com/camronwood/neural-junkie/internal/filechange	0.258s
ok  	github.com/camronwood/neural-junkie/internal/git	0.347s
ok  	github.com/camronwood/neural-junkie/internal/google/meetnotes	0.272s
ok  	github.com/camronwood/neural-junkie/internal/hardware	0.218s
ok  	github.com/camronwood/neural-junkie/internal/hfhub	3.871s
ok  	github.com/camronwood/neural-junkie/internal/hub	6.035s
ok  	github.com/camronwood/neural-junkie/internal/implementation/routing	0.248s
ok  	github.com/camronwood/neural-junkie/internal/integrations/slack	0.360s
ok  	github.com/camronwood/neural-junkie/internal/integrations/slack/relay	0.355s
ok  	github.com/camronwood/neural-junkie/internal/learning	0.274s
ok  	github.com/camronwood/neural-junkie/internal/lora/export	0.255s
ok  	github.com/camronwood/neural-junkie/internal/lora/train	0.223s
ok  	github.com/camronwood/neural-junkie/internal/lsp	0.243s
ok  	github.com/camronwood/neural-junkie/internal/mcp	0.814s
?   	github.com/camronwood/neural-junkie/internal/mcp/architecture	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/mcp/assistant	0.233s
?   	github.com/camronwood/neural-junkie/internal/mcp/backend	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/mcp/biology	0.220s
?   	github.com/camronwood/neural-junkie/internal/mcp/cad	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/mcp/codereview	0.230s
?   	github.com/camronwood/neural-junkie/internal/mcp/confluencemcp	[no test files]
?   	github.com/camronwood/neural-junkie/internal/mcp/database	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/mcp/devops	0.235s
ok  	github.com/camronwood/neural-junkie/internal/mcp/frontend	0.963s
?   	github.com/camronwood/neural-junkie/internal/mcp/repomcp	[no test files]
?   	github.com/camronwood/neural-junkie/internal/mcp/resources	[no test files]
?   	github.com/camronwood/neural-junkie/internal/mcp/rust	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/mcp/security	0.673s
ok  	github.com/camronwood/neural-junkie/internal/mcp/shared	0.208s
ok  	github.com/camronwood/neural-junkie/internal/mcp/workspace	0.308s
ok  	github.com/camronwood/neural-junkie/internal/mcp_export	0.288s
ok  	github.com/camronwood/neural-junkie/internal/memory	0.389s
ok  	github.com/camronwood/neural-junkie/internal/ollama	0.299s
ok  	github.com/camronwood/neural-junkie/internal/packs	0.246s
ok  	github.com/camronwood/neural-junkie/internal/pathutil	0.177s
--- FAIL: TestImportAnalysisIntegration (0.44s)
    import_integration_test.go:21: token refresh HTTP 403: {"error":"access_denied","error_description":"Unauthorized"}
--- FAIL: TestCheckStatusIntegration (0.17s)
    status_integration_test.go:24: expected authenticated: {Environment:dev CredentialsPath:/Users/camronwood/Library/Application Support/com.BrightestBio.bbio/credentials-dev.json Authenticated:false LoggedIn:false Identity: Hint:token refresh HTTP 403: {"error":"access_denied","error_description":"Unauthorized"}}
--- FAIL: TestListAnalysesIntegration (0.14s)
    status_integration_test.go:47: token refresh HTTP 403: {"error":"access_denied","error_description":"Unauthorized"}
FAIL
FAIL	github.com/camronwood/neural-junkie/internal/phoeniximport	1.036s
ok  	github.com/camronwood/neural-junkie/internal/protocol	0.234s
ok  	github.com/camronwood/neural-junkie/internal/repo	0.192s
ok  	github.com/camronwood/neural-junkie/internal/scananalysis	0.355s
ok  	github.com/camronwood/neural-junkie/internal/scansummary	0.254s
?   	github.com/camronwood/neural-junkie/internal/secondaryanalysis	[no test files]
?   	github.com/camronwood/neural-junkie/internal/store/sqlite	[no test files]
?   	github.com/camronwood/neural-junkie/internal/testutil	[no test files]
?   	github.com/camronwood/neural-junkie/internal/workspacefiles	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/workspacesymbols	0.193s
?   	github.com/camronwood/neural-junkie/scenarios/fixtures/minimal-repo/collabs/c4dc37b6-813d-4817-b9af-a3e4255494e4/core/sample	[no test files]
?   	github.com/camronwood/neural-junkie/scenarios/fixtures/minimal-repo/collabs/eedc0e58-4926-4d26-8b02-66b7eeea25dd/core/sample	[no test files]
?   	github.com/camronwood/neural-junkie/scenarios/fixtures/minimal-repo/core/obscure/internal	[no test files]
?   	github.com/camronwood/neural-junkie/scenarios/fixtures/minimal-repo/core/sample	[no test files]
?   	github.com/camronwood/neural-junkie/scripts	[no test files]
ok  	github.com/camronwood/neural-junkie/test	30.812s
FAIL
make[1]: *** [test-all] Error 1
```

