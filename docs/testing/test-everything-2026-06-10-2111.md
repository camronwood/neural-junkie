# test-everything — 2026-06-10-2111 UTC

- Hub: `http://127.0.0.1:18765`
- Full collab sweep (`FULL=1`): `False`
- Skip live: `True`
- Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration |
|-------|--------|----------|
| `test-all` | FAIL | 90s |

## Artifacts

- Full log: `/Users/camronwood/development/projects/neural-junkie/docs/testing/test-everything-2026-06-10-2111.log`

## Failures (tail)

### test-all (exit 2)

```text
🔍 go vet...

🧪 Go tests...
?   	github.com/camronwood/neural-junkie	[no test files]
?   	github.com/camronwood/neural-junkie/cmd/agent	[no test files]
?   	github.com/camronwood/neural-junkie/cmd/chat	[no test files]
?   	github.com/camronwood/neural-junkie/cmd/cli	[no test files]
ok  	github.com/camronwood/neural-junkie/cmd/server	2.143s
?   	github.com/camronwood/neural-junkie/cmd/slack-oauth-relay	[no test files]
?   	github.com/camronwood/neural-junkie/cmd/tool-approval-hook	[no test files]
?   	github.com/camronwood/neural-junkie/cmd/verify-bootstrap-lora	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/agent	1.472s
ok  	github.com/camronwood/neural-junkie/internal/ai	0.310s
ok  	github.com/camronwood/neural-junkie/internal/cad	0.264s
ok  	github.com/camronwood/neural-junkie/internal/chatcontext	0.213s
ok  	github.com/camronwood/neural-junkie/internal/cli	3.995s
ok  	github.com/camronwood/neural-junkie/internal/codeindex	0.239s
ok  	github.com/camronwood/neural-junkie/internal/collaboration	0.242s
ok  	github.com/camronwood/neural-junkie/internal/collaboration/actions	0.233s
ok  	github.com/camronwood/neural-junkie/internal/collaboration/routing	0.230s
ok  	github.com/camronwood/neural-junkie/internal/collabworktree	0.309s
ok  	github.com/camronwood/neural-junkie/internal/config	3.698s
ok  	github.com/camronwood/neural-junkie/internal/confluence	0.232s
ok  	github.com/camronwood/neural-junkie/internal/delegation	0.259s
ok  	github.com/camronwood/neural-junkie/internal/embed	0.280s
ok  	github.com/camronwood/neural-junkie/internal/filechange	0.236s
ok  	github.com/camronwood/neural-junkie/internal/git	0.351s
ok  	github.com/camronwood/neural-junkie/internal/google/meetnotes	0.319s
ok  	github.com/camronwood/neural-junkie/internal/hardware	0.247s
--- FAIL: TestCatalogLocalDownloadURLsReachable (11.42s)
    catalog_test.go:86: TheBloke/deepseek-coder-6.7b-instruct-GGUF: Head "https://us.aws.cdn.hf.co/xet-bridge-us/65479a438767484a0548595b/c60ba218901d1adcbe6990455c93db103929a800a6682b3f819f34967fa84553?response-content-disposition=inline%3B+filename*%3DUTF-8%27%27deepseek-coder-6.7b-instruct.Q4_K_M.gguf%3B+filename%3D%22deepseek-coder-6.7b-instruct.Q4_K_M.gguf%22%3B&X-Xet-Cas-Uid=public&Expires=1781129492&Policy=eyJTdGF0ZW1lbnQiOlt7IlJlc291cmNlIjoiaHR0cHM6Ly91cy5hd3MuY2RuLmhmLmNvL3hldC1icmlkZ2UtdXMvNjU0NzlhNDM4NzY3NDg0YTA1NDg1OTViL2M2MGJhMjE4OTAxZDFhZGNiZTY5OTA0NTVjOTNkYjEwMzkyOWE4MDBhNjY4MmIzZjgxOWYzNDk2N2ZhODQ1NTNcXD9yZXNwb25zZS1jb250ZW50LWRpc3Bvc2l0aW9uPWlubGluZSUzQitmaWxlbmFtZSUyQSUzRFVURi04JTI3JTI3ZGVlcHNlZWstY29kZXItNi43Yi1pbnN0cnVjdC5RNF9LX00uZ2d1ZiUzQitmaWxlbmFtZSUzRCUyMmRlZXBzZWVrLWNvZGVyLTYuN2ItaW5zdHJ1Y3QuUTRfS19NLmdndWYlMjIlM0ImWC1YZXQtQ2FzLVVpZD1wdWJsaWMiLCJDb25kaXRpb24iOnsiRGF0ZUxlc3NUaGFuIjp7IkVwb2NoVGltZSI6MTc4MTEyOTQ5Mn19fV19&Signature=MEQCICgtU6YSpeXK%7Es0Whd9V99dhZ0QKeHfTfgjBB5cMGV5qAiAFb9iskbIbMsIfvI%7EXBFKdZLnKPvBpmJWs5hm9kxFLog__&Key-Pair-Id=01KAYHXK2CBJSW0YZTMNXK9W1M": net/http: TLS handshake timeout
FAIL
FAIL	github.com/camronwood/neural-junkie/internal/hfhub	11.699s
ok  	github.com/camronwood/neural-junkie/internal/hub	7.654s
ok  	github.com/camronwood/neural-junkie/internal/implementation/routing	0.239s
ok  	github.com/camronwood/neural-junkie/internal/integrations/slack	0.341s
ok  	github.com/camronwood/neural-junkie/internal/integrations/slack/relay	0.309s
ok  	github.com/camronwood/neural-junkie/internal/learning	0.293s
ok  	github.com/camronwood/neural-junkie/internal/lora/export	0.271s
ok  	github.com/camronwood/neural-junkie/internal/lora/train	0.246s
ok  	github.com/camronwood/neural-junkie/internal/lsp	0.182s
ok  	github.com/camronwood/neural-junkie/internal/mcp	0.718s
?   	github.com/camronwood/neural-junkie/internal/mcp/architecture	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/mcp/assistant	0.317s
?   	github.com/camronwood/neural-junkie/internal/mcp/backend	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/mcp/biology	0.340s
?   	github.com/camronwood/neural-junkie/internal/mcp/cad	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/mcp/codereview	0.218s
?   	github.com/camronwood/neural-junkie/internal/mcp/confluencemcp	[no test files]
?   	github.com/camronwood/neural-junkie/internal/mcp/database	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/mcp/devops	0.221s
ok  	github.com/camronwood/neural-junkie/internal/mcp/frontend	0.650s
?   	github.com/camronwood/neural-junkie/internal/mcp/repomcp	[no test files]
?   	github.com/camronwood/neural-junkie/internal/mcp/resources	[no test files]
?   	github.com/camronwood/neural-junkie/internal/mcp/rust	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/mcp/security	0.630s
ok  	github.com/camronwood/neural-junkie/internal/mcp/shared	0.234s
ok  	github.com/camronwood/neural-junkie/internal/mcp/workspace	0.252s
ok  	github.com/camronwood/neural-junkie/internal/mcp_export	0.234s
ok  	github.com/camronwood/neural-junkie/internal/memory	0.323s
ok  	github.com/camronwood/neural-junkie/internal/ollama	0.279s
ok  	github.com/camronwood/neural-junkie/internal/packs	0.250s
ok  	github.com/camronwood/neural-junkie/internal/pathutil	0.202s
--- FAIL: TestImportAnalysisIntegration (0.58s)
    import_integration_test.go:21: token refresh HTTP 403: {"error":"access_denied","error_description":"Unauthorized"}
--- FAIL: TestCheckStatusIntegration (0.13s)
    status_integration_test.go:24: expected authenticated: {Environment:dev CredentialsPath:/Users/camronwood/Library/Application Support/com.BrightestBio.bbio/credentials-dev.json Authenticated:false LoggedIn:false Identity: Hint:token refresh HTTP 403: {"error":"access_denied","error_description":"Unauthorized"}}
--- FAIL: TestListAnalysesIntegration (0.13s)
    status_integration_test.go:47: token refresh HTTP 403: {"error":"access_denied","error_description":"Unauthorized"}
FAIL
FAIL	github.com/camronwood/neural-junkie/internal/phoeniximport	1.071s
ok  	github.com/camronwood/neural-junkie/internal/protocol	0.299s
ok  	github.com/camronwood/neural-junkie/internal/repo	0.217s
ok  	github.com/camronwood/neural-junkie/internal/scananalysis	0.366s
ok  	github.com/camronwood/neural-junkie/internal/scansummary	0.267s
?   	github.com/camronwood/neural-junkie/internal/secondaryanalysis	[no test files]
?   	github.com/camronwood/neural-junkie/internal/store/sqlite	[no test files]
?   	github.com/camronwood/neural-junkie/internal/testutil	[no test files]
?   	github.com/camronwood/neural-junkie/internal/workspacefiles	[no test files]
ok  	github.com/camronwood/neural-junkie/internal/workspacesymbols	0.217s
?   	github.com/camronwood/neural-junkie/scenarios/fixtures/minimal-repo/collabs/c4dc37b6-813d-4817-b9af-a3e4255494e4/core/sample	[no test files]
?   	github.com/camronwood/neural-junkie/scenarios/fixtures/minimal-repo/collabs/eedc0e58-4926-4d26-8b02-66b7eeea25dd/core/sample	[no test files]
?   	github.com/camronwood/neural-junkie/scenarios/fixtures/minimal-repo/core/obscure/internal	[no test files]
?   	github.com/camronwood/neural-junkie/scenarios/fixtures/minimal-repo/core/sample	[no test files]
?   	github.com/camronwood/neural-junkie/scripts	[no test files]
ok  	github.com/camronwood/neural-junkie/test	30.997s
FAIL
make[1]: *** [test-all] Error 1
```

