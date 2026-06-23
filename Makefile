.PHONY: help build run-server run-agents run-all demo clean docs stop refresh test test-go test-all test-messages slack-vendor-check slack-vendor-json gallery-sync articles-sync deps-lora server-regression server-debug collab-scenarios-all collab-preflight slack-smoke test-regression-live chat-scenarios-debug test-parity-stable test-parity-stable-restart test-parity-full-restart parity-scenarios parity-scenarios-list test-regression-bundle test-conversation-contract test-everything test-everything-full release-prep slack-oauth-relay-deploy-cf slack-oauth-relay-deploy

# Bundled Neural Junkie Slack app (maintainer: ../../sandbox/scripts/slack-creds-to-vendor.sh)
SLACK_VENDOR_JSON := internal/integrations/slack/vendor/oauth.json
GOOGLE_VENDOR_JSON := internal/google/meetnotes/vendor/oauth.json
SERVER_GO_BUILD_TAGS :=
ifneq (,$(wildcard $(SLACK_VENDOR_JSON)))
SERVER_GO_BUILD_TAGS += slackvendor
endif
ifneq (,$(wildcard $(GOOGLE_VENDOR_JSON)))
SERVER_GO_BUILD_TAGS += googlevendor
endif
ifneq (,$(strip $(SERVER_GO_BUILD_TAGS)))
SERVER_GO_TAGS := -tags="$(strip $(SERVER_GO_BUILD_TAGS))"
else
SERVER_GO_TAGS :=
endif

help: ## Show this help
	@echo "Neural Junkie - Multi-Agent Collaboration System"
	@echo ""
	@echo "Quick Start: make gui  (first time: make gui-install)"
	@echo "Documentation: make docs"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

docs: ## Show documentation guide
	@cat DOCS.md

slack-vendor-json: ## Generate gitignored vendor/oauth.json from sandbox scripts/.slack-creds
	@bash ../../sandbox/scripts/slack-creds-to-vendor.sh

slack-vendor-check: ## Fail if vendor/oauth.json is missing (release / CI)
	@test -f $(SLACK_VENDOR_JSON) || (echo "❌ Missing $(SLACK_VENDOR_JSON) — run: make slack-vendor-json" >&2; exit 1)
	@echo "✅ $(SLACK_VENDOR_JSON) present"

slack-oauth-relay-deploy-cf: ## Deploy public HTTPS Slack OAuth relay to Cloudflare Workers (free)
	@./scripts/deploy-slack-oauth-relay-cloudflare.sh

slack-oauth-relay-deploy: ## Deploy public HTTPS Slack OAuth relay to AWS Lambda (requires AWS SSO)
	@AWS_PROFILE=$${AWS_PROFILE:-AdministratorAccess-566982197870} ./scripts/deploy-slack-oauth-relay-aws.sh

gallery-sync: ## Copy ads/screenshots to docs/media/gallery and rebuild manifest
	@chmod +x ./scripts/sync-gallery.sh
	@./scripts/sync-gallery.sh

articles-sync: ## Regenerate docs/articles from docs/marketing LinkedIn sources
	@chmod +x ./scripts/sync-articles.sh
	@./scripts/sync-articles.sh

build: ## Build all binaries
	@echo "🔨 Building server... $(if $(SERVER_GO_TAGS),[Slack vendor embedded],)"
	@go build $(SERVER_GO_TAGS) -o bin/server ./cmd/server
	@echo "🔨 Building agent runner..."
	@go build -o bin/agent cmd/agent/main.go
	@echo "🔨 Building CLI..."
	@go build -o bin/cli cmd/cli/main.go
	@echo "🔨 Building interactive chat..."
	@go build -o bin/chat cmd/chat/main.go
	@echo "✅ Build complete!"
	@echo ""
	@echo "💡 For GUI, run: make gui-build"

run-server: ## Start the chat hub server
	@echo "🚀 Starting chat hub server on http://localhost:18765 $(if $(SERVER_GO_TAGS),[Slack vendor],)"
	@go run $(SERVER_GO_TAGS) ./cmd/server

server: setup-env ## Start server with environment loaded
	@echo "🚀 Starting chat hub server with environment from env.local... $(if $(SERVER_GO_TAGS),[Slack vendor],)"
	@bash -c 'source load-env.sh && go run $(SERVER_GO_TAGS) ./cmd/server'

server-debug: setup-env ## Hub with NEURAL_JUNKIE_DEBUG=1 (pprof + /api/debug/hub-memory); logs to /tmp/nj-hub.log
	@echo "🔧 Starting debug hub → /tmp/nj-hub.log  (pprof: http://127.0.0.1:6060/debug/pprof/) $(if $(SERVER_GO_TAGS),[Slack vendor],)"
	@bash -c 'source load-env.sh && NEURAL_JUNKIE_DEBUG=1 go run $(SERVER_GO_TAGS) ./cmd/server 2>&1 | tee /tmp/nj-hub.log'

server-regression: setup-env ## Hub for live scenario regression (RATE_LIMIT=0 + DEBUG=1); logs to /tmp/nj-hub.log
	@echo "🔧 Regression hub → /tmp/nj-hub.log  (NEURAL_JUNKIE_RATE_LIMIT=0 NEURAL_JUNKIE_DEBUG=1 NEURAL_JUNKIE_SLACK_DISABLED=1) $(if $(SERVER_GO_TAGS),[Slack vendor],)"
	@bash -c 'source load-env.sh && NEURAL_JUNKIE_RATE_LIMIT=0 NEURAL_JUNKIE_DEBUG=1 NEURAL_JUNKIE_SLACK_DISABLED=1 go run $(SERVER_GO_TAGS) ./cmd/server 2>&1 | tee /tmp/nj-hub.log'

server-log: ## Tail collab-related lines from /tmp/nj-hub.log (run server-debug first)
	@python3 scripts/debug-collab.py watch --log /tmp/nj-hub.log

debug-session: ## Analyze ~/.neural-junkie/last-session.json (collabs, joins, channels); conversation issues: python3 scripts/analyze-session-conversation-issues.py [--self-test] (keys: premature_file_apply_claim, missing_prior_reference, file_export_chat_mode, placeholder_proposals, stale_session_summary, absolute_path_in_chat)
	@python3 scripts/debug-collab.py session

debug-collab: ## Live collab state from hub (optional: CHANNEL=... COLAB=ec2cdef8)
	@python3 scripts/debug-collab.py live \
		$(if $(CHANNEL),--channel $(CHANNEL),) \
		$(if $(COLAB),--collab $(COLAB),) \
		--include-terminal

debug-messages: ## Last messages for CHANNEL (session file; add LIVE=1 for hub)
	@if [ -z "$(CHANNEL)" ]; then echo "Usage: make debug-messages CHANNEL=dm-camron-assistant [TAIL=20] [LIVE=1]"; exit 1; fi
	@python3 scripts/debug-collab.py messages --channel "$(CHANNEL)" --tail $(if $(TAIL),$(TAIL),20) \
		$(if $(LIVE),--live,)

collab-smoke: ## Collab lifecycle smoke (API phases); LIVE=1 for running hub
	@python3 scripts/collab-smoke.py $(if $(LIVE),--live,)

slack-smoke: ## Slack integration + hub /api/slack handler smoke (CI-safe; LIVE=1 runs slack-live-smoke.sh)
	@go test ./internal/integrations/slack/... -count=1
	@go test ./cmd/server -run 'TestHandleSlack' -count=1
	@if [ "$(LIVE)" = "1" ]; then \
		chmod +x scripts/slack-live-smoke.sh; \
		./scripts/slack-live-smoke.sh; \
	fi

collab-preflight: ## Fail-fast checks before collab-scenarios-all (hub, Ollama, agents, scenario list)
	@bash -c 'source load-env.sh && python3 scripts/collab-preflight.py $(if $(REQUIRE_GEMINI),--require-gemini,)'

test-regression-live: ## Print pre-release live regression checklist (does not start hub)
	@echo "Pre-release live regression (see docs/TESTING.md):"
	@echo ""
	@echo "  make release-prep                 # one-shot: env+Gemini judge + test-everything-full + parity + benchmark"
	@echo "  make test-everything              # CI + live harness; review docs/testing/test-everything-*.md"
	@echo "  make test-everything-full         # above + all collab scenarios (~1-3h extra)"
	@echo ""
	@echo "  0. ollama serve  &&  make pull-benchmark-models   # quick suite, ≤24B models"
	@echo "  0b. make release-prep             # auto-loads .gemini-api-key, verifies hub judge, runs full gate"
	@echo "  1. make server-regression     # hub: RATE_LIMIT=0 + DEBUG=1"
	@echo "  2. Agents online (specialists + Gemini for resource-api-schema-planning)"
	@echo "  3. make test-regression-bundle   # implement + chat + conversation (~30-60m)"
	@echo "  4. make test-parity-stable-restart  # optional: 3x implement with hub restart"
	@echo "  5. make chat-scenarios-debug"
	@echo "  6. make collab-scenarios-all  # ~1-3h serial"
	@echo "  7. make learning-scenarios"
	@echo "  Optional: LIVE=1 make slack-smoke, collab matrices, NEURAL_JUNKIE_SCENARIO_REPO=..."

learning-lora-smoke: ## Personal learning + LoRA expert-context smoke (CI, no GPU)
	@go test ./cmd/server/ -run TestLearningLoRASmoke -count=1
	@go test ./internal/learning/... -count=1

learning-scenario: ## Run one live learning scenario (SCENARIO=learning-save-and-list)
	@if [ -z "$(SCENARIO)" ]; then echo "Usage: make learning-scenario SCENARIO=learning-save-and-list [VERBOSE=1]"; exit 1; fi
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/learning-scenarios.py --scenario "$(SCENARIO)" \
		$(if $(VERBOSE),--verbose,)

learning-scenarios: ## Run all live learning scenarios under scenarios/learning/
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/learning-scenarios.py --all \
		$(if $(VERBOSE),--verbose,)

collab-scenario: ## Run one live collab scenario (SCENARIO=planning-two-agent, PROFILE=fast|realistic, KEEP=1)
	@if [ -z "$(SCENARIO)" ]; then echo "Usage: make collab-scenario SCENARIO=planning-two-agent [PROFILE=fast] [KEEP=1] [VERBOSE=1]"; exit 1; fi
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --scenario "$(SCENARIO)" \
		$(if $(PROFILE),--profile $(PROFILE),) \
		$(if $(VERBOSE),--verbose,) \
		$(if $(KEEP),--keep,)

collab-scenarios: ## Run all live collab scenarios (hub should use make server-regression)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --all \
		$(if $(PROFILE),--profile $(PROFILE),) \
		$(if $(VERBOSE),--verbose,)

collab-scenarios-all: collab-scenarios ## Alias: full collab sweep (15 scenarios; PROFILE does not shorten timeouts)

collab-sweep-serial: ## Run collab scenarios one-by-one; stop on FAIL (RESUME=1 skips PASS in docs/testing/collab-matrix.tsv)
	@chmod +x scripts/collab-sweep-serial.sh
	@RETRIES=$(or $(RETRIES),1) RESUME=$(RESUME) ONLY="$(ONLY)" VERBOSE=$(VERBOSE) \
		./scripts/collab-sweep-serial.sh

collab-failure-repro: ## Serial repro for known collab failure scenarios (RESTART_HUB=1 VERBOSE=1)
	@chmod +x scripts/collab-failure-repro.sh
	@RESTART_HUB=$(or $(RESTART_HUB),0) VERBOSE=$(VERBOSE) STOP_ON_FAIL=$(or $(STOP_ON_FAIL),1) \
		./scripts/collab-failure-repro.sh $(ONLY)

collab-scenario-matrix: ## Sweep agent profiles and round budgets (planning-two-agent template)
	@chmod +x scripts/collab-scenario-matrix.sh
	@./scripts/collab-scenario-matrix.sh

collab-routing-matrix: ## A/B smart routing on execute-deliverable (needs live hub + agents)
	@chmod +x scripts/collab-routing-matrix.sh
	@NEURAL_JUNKIE_RATE_LIMIT=0 ./scripts/collab-routing-matrix.sh

collab-parity: ## Solo vs collab deliverable parity on minimal-repo fixture
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --scenario solo-vs-collab-parity \
		$(if $(PROFILE),--profile $(PROFILE),--profile fast,) \
		$(if $(VERBOSE),--verbose,)

collab-scenario-regression: ## Run collab edge-case regression scenarios (plan parser + execution guards)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --scenario plan-dependency-prose-regression $(if $(VERBOSE),--verbose,)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --scenario plan-findings-task-regression $(if $(VERBOSE),--verbose,)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --scenario plan-distinct-deliverables-same-agent $(if $(VERBOSE),--verbose,)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --scenario document-findings-execution $(if $(VERBOSE),--verbose,)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --scenario execution-no-stack-commands $(if $(VERBOSE),--verbose,)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression $(if $(VERBOSE),--verbose,)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --scenario collab-no-edit-after-cancel $(if $(VERBOSE),--verbose,)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --scenario collaboration-station-website $(if $(VERBOSE),--verbose,)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --scenario collaboration-station-website-sa $(if $(VERBOSE),--verbose,)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --scenario make-me-a-website $(if $(VERBOSE),--verbose,)

conversation-scenarios-regression: ## Chat workspace + collab conversation quality (session-issue guards)
	@chmod +x scripts/conversation-scenarios-regression.py
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/conversation-scenarios-regression.py $(if $(VERBOSE),--verbose,)

test-collab-plan: ## Deterministic Go tests for collab plan parsing regressions (CI-safe)
	@go test ./internal/collaboration/... ./internal/hub/... -count=1 -run 'Regression|DependencyProse|Findings|4ea36409|f7518f88|DocumentFindings|DistinctDeliverable|StackTool|FilterCollab|SuppressMCP'

test-conversation-contract: ## CI-safe conversation + collab wiring contract (agent, hub, desktop, smoke)
	@echo "🧪 Agent conversation routing..."
	@go test ./internal/agent/ -count=1 -run 'ChatQuality|ConversationalClosure|ConversationMode|CollabGeneration|TurnIntent'
	@echo "🧪 Hub conversation/collab wiring..."
	@go test ./internal/hub/ -count=1 -run 'Collab|ChannelHold|Interject|DMMention|SessionCollab|CreateDM|MultiTurn'
	@go test ./cmd/server/ -count=1 -run 'ChannelInterject'
	@echo "🧪 Collab API smoke..."
	@go test ./test/ -count=1 -run 'TestCollabSmokePhaseTransitions'
	@echo "🧪 Scenario assertion helpers..."
	@$(MAKE) test-scenario-assert
	@echo "🧪 Desktop chat/collab Vitest..."
	@cd desktop && npx vitest run \
	  src/stores/chatStreamCoalesce.test.ts \
	  src/stores/chatStoreChannelHold.test.ts \
	  src/stores/chatStreamReasoning.test.ts \
	  src/utils/outboundChatMetadata.test.ts \
	  src/utils/prepareOutboundPayload.test.ts \
	  src/utils/conversationMode.test.ts \
	  src/utils/collaborationPanelState.test.ts \
	  src/utils/collaborationConfirm.test.ts \
	  src/utils/collaborationTaskOrchestration.test.ts \
	  src/components/ChatWindow.collaboration.test.tsx \
	  src/components/ChatWindow.interject.test.tsx \
	  src/components/CollaborationPanel.test.tsx

test-scenario-assert: ## Python unit tests for scenario assertion + deliverable contracts
	@cd scripts/lib && PYTHONPATH=.. python3 -m unittest scenario_assert_test.py scenario_contract_test.py collab_hub_test.py hub_regression_test.py
	@PYTHONPATH=scripts python3 scripts/lib/scenario_contract.py

chat-scenario: ## Run one live chat scenario (SCENARIO=greeting-chat-mode, KEEP=1)
	@if [ -z "$(SCENARIO)" ]; then echo "Usage: make chat-scenario SCENARIO=greeting-chat-mode [VERBOSE=1] [KEEP=1]"; exit 1; fi
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/chat-scenarios.py --scenario "$(SCENARIO)" \
		$(if $(VERBOSE),--verbose,) \
		$(if $(KEEP),--keep,)

chat-scenarios: ## Run all live chat scenarios under scenarios/chat/
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/chat-scenarios.py --all \
		$(if $(VERBOSE),--verbose,)

chat-scenarios-dm: ## Run DM chat scenarios only (--tag dm)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/chat-scenarios.py --all --tag dm \
		$(if $(VERBOSE),--verbose,)

chat-scenarios-regression: ## Run regression-tagged chat scenarios (workspace, echo, closure)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/chat-scenarios.py --all --tag regression \
		$(if $(VERBOSE),--verbose,)

chat-scenarios-debug: ## Run debug-tagged chat scenarios (requires hub: make server-regression)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/chat-scenarios.py --all --tag debug --require-debug \
		$(if $(VERBOSE),--verbose,)

chat-scenarios-list: ## List chat scenarios and tags
	@python3 scripts/chat-scenarios.py --list

implement-scenario: ## Run one implementation scenario (SCENARIO=go-handler)
	@if [ -z "$(SCENARIO)" ]; then echo "Usage: make implement-scenario SCENARIO=go-handler"; exit 1; fi
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/implement-scenarios.py --scenario "$(SCENARIO)" \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}" $(if $(KEEP),--keep,)

implement-scenarios: ## Run all scenarios under scenarios/implement/
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/implement-scenarios.py --all \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}" $(if $(KEEP),--keep,)

implement-scenarios-list: ## List implementation scenarios
	@python3 scripts/implement-scenarios.py --list

parity-scenario: ## Run one parity scenario (SCENARIO=large-repo-semantic-find)
	@if [ -z "$(SCENARIO)" ]; then echo "Usage: make parity-scenario SCENARIO=large-repo-semantic-find"; exit 1; fi
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/parity-scenarios.py --scenario "$(SCENARIO)" \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}" $(if $(KEEP),--keep,)

parity-scenarios: ## Run all scenarios under scenarios/parity/
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/parity-scenarios.py --all \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}" $(if $(KEEP),--keep,)

parity-scenarios-list: ## List Cursor parity scenarios
	@python3 scripts/parity-scenarios.py --list

test-parity-full-restart: ## implement + parity scenarios 3× with hub restart between sweeps
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/parity-full-restart.py --runs 3 \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"

test-parity-stable: ## Run implement-scenarios 3x with hub restart between sweeps (stable gate)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/implement-scenarios-stable.py --runs 3 --min-pass 16 \
		--restart-between --hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"

test-parity-stable-stress: ## Run implement-scenarios 3x back-to-back (may OOM hub on tight memory)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/implement-scenarios-stable.py --runs 3 --min-pass 16 \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"

test-parity-stable-restart: test-parity-stable ## Alias for stable gate (restart between sweeps)

test-regression-bundle: ## Live bundle: implement + chat-regression + conversation-regression
	@chmod +x scripts/regression-bundle.py
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/regression-bundle.py \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}" $(if $(VERBOSE),--verbose,)

test-everything: ## CI smoke + live harness; writes docs/testing/test-everything-*.md (SKIP_LIVE=1 CI-only; FULL=1 all collab)
	@chmod +x scripts/test-everything.py
	@python3 scripts/test-everything.py \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}" \
		$(if $(FULL),--full,) \
		$(if $(VERBOSE),--verbose,) \
		$(if $(SKIP_LIVE),--skip-live,) \
		$(if $(CONTINUE),--continue-on-fail,)

test-everything-full: ## test-everything with FULL=1 (includes collab-scenarios-all, ~1-3h)
	@$(MAKE) test-everything FULL=1 $(if $(VERBOSE),VERBOSE=1,) $(if $(CONTINUE),CONTINUE=1,)

release-prep: ## Full release gate: test-everything-full + parity-restart + benchmark → docs/testing/release-prep-*.md
	@chmod +x scripts/release-prep.py scripts/test-everything.py
	@bash -c 'source load-env.sh && NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/release-prep.py \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}" \
		$(if $(SKIP_LIVE),--skip-live,) \
		$(if $(NO_FULL),--no-full,) \
		$(if $(SKIP_PARITY),--skip-parity,) \
		$(if $(SKIP_BENCHMARK),--skip-benchmark,) \
		$(if $(SKIP_EVERYTHING),--skip-everything,) \
		$(if $(BENCHMARK_SUITE),--benchmark-suite $(BENCHMARK_SUITE),) \
		$(if $(BENCHMARK_MODELS),--benchmark-models "$(BENCHMARK_MODELS)",) \
		$(if $(NO_PULL),--no-pull-models,) \
		$(if $(BENCHMARK_ALLOW_LARGE),--benchmark-allow-large,) \
		$(if $(NO_RESTART_HUB),--no-restart-hub,) \
		$(if $(VERBOSE),--verbose,) \
		$(if $(STOP_ON_FAIL),--stop-on-fail,)'

model-benchmark: ## Benchmark ≤24B coder models (SUITE=quick; pulls by default; NO_PULL=1 to skip; BENCHMARK_ALLOW_LARGE=1 to bypass cap)
	@chmod +x scripts/model-benchmark-suite.py
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/model-benchmark-suite.py \
		--suite $(or $(SUITE),quick) \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}" \
		$(if $(NO_PULL),,--pull) \
		$(if $(SKIP_MISSING),--skip-missing,) \
		$(if $(MODELS),--models "$(MODELS)",) \
		$(if $(BENCHMARK_ALLOW_LARGE),--allow-large-models,) \
		$(if $(VERBOSE),--verbose,)

pull-benchmark-models: ## Pull Ollama models for benchmark suite (SUITE=quick; PULL_ALL=1 to bypass cap)
	@chmod +x scripts/pull-benchmark-models.py
	@python3 scripts/pull-benchmark-models.py \
		--suite $(or $(SUITE),quick) \
		$(if $(PULL_ALL),--allow-large-models,) \
		$(if $(MODELS),--models "$(MODELS)",)

model-benchmark-list: ## List benchmark suites and default model roster
	@python3 scripts/model-benchmark-suite.py --list-suites
	@echo ""
	@python3 scripts/model-benchmark-suite.py --list-models

publish-model-benchmarks: ## Merge docs/testing/model-benchmark-*.json → docs/data for website
	@chmod +x scripts/publish-model-benchmarks.py
	@python3 scripts/publish-model-benchmarks.py

chat: ## Start interactive chat client
	@echo "💬 Starting interactive chat client..."
	@go run cmd/chat/main.go

ensure-ollama: ## Start Ollama when not healthy (system PATH or fetch-ollama bundle)
	@chmod +x scripts/ensure-ollama.sh
	@./scripts/ensure-ollama.sh

ensure-ollama-bundle: ## Fetch Ollama runtime for Tauri bundle when missing (first-time desktop dev)
	@triple=$$(rustc -vV | grep host | cut -d' ' -f2); \
	bundle="desktop/src-tauri/ollama/$$triple"; \
	if [ -f "$$bundle/ollama" ] || [ -f "$$bundle/bin/ollama" ]; then \
		exit 0; \
	fi; \
	echo "📥 Fetching Ollama bundle for Tauri ($$triple — first time, may take a minute)..."; \
	chmod +x scripts/fetch-ollama.sh; \
	./scripts/fetch-ollama.sh

gui: ensure-sidecar ensure-ollama-bundle ensure-ollama ensure-lora-deps ## Start GUI desktop app (Tauri + React)
	@echo "🖥️  Starting desktop app with React..."
	@cd desktop && npm run tauri:dev

ensure-sidecar: ## Build sidecar binary if missing (needed for Tauri dev)
	@if [ ! -f desktop/src-tauri/binaries/nj-server-$$(rustc -vV | grep host | cut -d' ' -f2) ]; then \
		echo "🔨 Building sidecar binary for dev..."; \
		$(MAKE) build-sidecar; \
	fi

gui-install: ensure-lora-deps ## Install GUI dependencies (first time only)
	@echo "📦 Installing desktop app dependencies..."
	@cd desktop && npm install
	@echo "✅ Desktop dependencies installed!"

gui-build: ## Build production desktop app
	@echo "🔨 Building desktop app..."
	@cd desktop && npm run tauri:build
	@echo "✅ Desktop app built! Check desktop/src-tauri/target/release/bundle/"

# Desktop aliases (for documentation consistency)
desktop: gui ## Alias for 'make gui'

desktop-install: gui-install ## Alias for 'make gui-install'

desktop-build: gui-build ## Alias for 'make gui-build'


test-messages: ## Test message sending functionality
	@./scripts/test-message-sending.sh

test-go: ## Run Go unit tests only (repeatable: -count=1)
	@echo "🧪 Running Go unit tests..."
	@go test ./... -count=1 -p 1
	@chmod +x ./scripts/cleanup-test-artifacts.py
	@./scripts/cleanup-test-artifacts.py || true
	@echo "✅ Go tests complete."

test-all: ## Run go vet, Go tests, desktop tsc, and Vitest (full CI-style)
	@echo "🔍 go vet..."
	@go vet ./...
	@echo ""
	@echo "🧪 Go tests..."
	@go test ./... -count=1 -p 1
	@echo ""
	@echo "🧪 Desktop typecheck (tsc)..."
	@cd desktop && npx tsc --noEmit
	@echo ""
	@echo "🧪 Desktop unit tests (Vitest)..."
	@cd desktop && npm test
	@echo ""
	@echo "🧪 Desktop coverage summary (Vitest)..."
	@cd desktop && npm run test:coverage
	@echo ""
	@echo "✅ Full test pass complete (vet + Go + desktop tsc + Vitest + coverage)."

test-race: ## Run Go race detector on concurrency-sensitive packages
	@echo "🧪 Go race detector (hub + filechange)..."
	@go test -race -count=1 ./internal/hub/... ./internal/filechange/...
	@echo "✅ Race tests complete."

demo-messages: ## Send demo messages to test the system
	@./scripts/demo-messages.sh

run-agents: ## Start all agent types
	@echo "🤖 Starting agents..."
	@go run cmd/agent/main.go --type backend --name "BackendEngineer" --channel general &
	@sleep 2
	@go run cmd/agent/main.go --type frontend --name "FrontendEngineer" --channel general &
	@sleep 2
	@go run cmd/agent/main.go --type devops --name "PlatformEngineer" --channel general &
	@sleep 2
	@go run cmd/agent/main.go --type security --name "SecurityReviewer" --channel general &
	@sleep 2
	@go run cmd/agent/main.go --type architecture --name "SoftwareArchitect" --channel general &
	@sleep 2
	@go run cmd/agent/main.go --type code-review --name "CodeReviewer" --channel general &
	@echo "✅ All agents started!"

# Individual agent targets with environment loaded
agent-backend: setup-env ## Start backend engineer agent
	@echo "🤖 Starting Backend Engineer..."
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type backend --name "BackendEngineer"'

agent-frontend: setup-env ## Start frontend engineer agent
	@echo "🤖 Starting Frontend Engineer..."
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type frontend --name "FrontendEngineer"'

agent-database: setup-env ## Start legacy database specialist agent
	@echo "🤖 Starting Database Specialist..."
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type database --name "DatabaseSpecialist"'

agent-security: setup-env ## Start security reviewer agent
	@echo "🤖 Starting Security Reviewer..."
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type security --name "SecurityReviewer"'

agent-devops: setup-env ## Start platform engineer agent
	@echo "🤖 Starting Platform Engineer..."
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type devops --name "PlatformEngineer"'

agent-architecture: setup-env ## Start software architect agent
	@echo "🤖 Starting Software Architect..."
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type architecture --name "SoftwareArchitect"'

agent-code-review: setup-env ## Start code reviewer agent
	@echo "🤖 Starting Code Reviewer..."
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type code-review --name "CodeReviewer"'

agents: setup-env ## Start all agents with environment loaded
	@echo "🤖 Starting all agents with environment from env.local..."
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type backend --name "BackendEngineer" --model "$${OLLAMA_CODE_MODEL:-qwen3.5:9b}" &'
	@sleep 2
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type frontend --name "FrontendEngineer" --model "$${OLLAMA_CODE_MODEL:-qwen3.5:9b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type devops --name "PlatformEngineer" --model "$${OLLAMA_CODE_MODEL:-qwen3.5:9b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type security --name "SecurityReviewer" --model "$${OLLAMA_CODE_MODEL:-qwen3.5:9b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type architecture --name "SoftwareArchitect" --model "$${OLLAMA_CODE_MODEL:-qwen3.5:9b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type code-review --name "CodeReviewer" --model "$${OLLAMA_CODE_MODEL:-qwen3.5:9b}" &'
	@echo "✅ All agents started!"

stop: ## Stop all running processes (server, agents, GUI)
	@echo "🛑 Stopping all Neural Junkie processes..."
	@bash -c 'cd "$(CURDIR)"; \
		HUB_PORT=18765; \
		if [ -f env.local ]; then \
			v=$$(grep -E "^[[:space:]]*SERVER_PORT=" env.local | tail -1 | cut -d= -f2- | tr -d "\r" | tr -d " "); \
			[ -n "$$v" ] && HUB_PORT=$$v; \
		fi; \
		lsof -ti :$$HUB_PORT 2>/dev/null | xargs kill -9 2>/dev/null || true; \
		lsof -ti :18765 2>/dev/null | xargs kill -9 2>/dev/null || true'
	@lsof -ti :1420 2>/dev/null | xargs kill -9 2>/dev/null || true
	@pkill -f "go run ./cmd/server" 2>/dev/null || pkill -f "cmd/server/main.go" 2>/dev/null || true
	@pkill -x server 2>/dev/null || pkill -f "$(CURDIR)/bin/server" 2>/dev/null || true
	@pkill -f "cmd/agent/main.go" 2>/dev/null || true
	@pkill -f "tauri dev" 2>/dev/null || true
	@pkill -f "Neural Junkie" 2>/dev/null || true
	@echo "✅ All processes stopped!"

refresh: stop setup-env ## Refresh: stop everything, clear logs, and restart fresh
	@echo ""
	@echo "🔄 Refreshing Neural Junkie system..."
	@echo "📝 Clearing logs..."
	@rm -f /tmp/chat-server.log 2>/dev/null || true
	@sleep 2
	@echo ""
	@echo "🚀 Starting server with fresh state..."
	@echo "   (Specialist agents are started in-process by the server via config)"
	@bash -c 'source load-env.sh && go run $(SERVER_GO_TAGS) ./cmd/server > /tmp/chat-server.log 2>&1 &'
	@sleep 3
	@echo ""
	@echo "✅ System refreshed! All processes restarted with clean state."
	@echo "📊 Server logs: /tmp/chat-server.log"
	@echo ""
	@echo "🖥️  To open GUI, run: make gui"
	@echo ""

start-all: setup-env ensure-ollama ensure-lora-deps ## Start server and all agents with environment loaded
	@bash -c 'cd "$(CURDIR)"; \
		source ./load-env.sh; \
		PORT="$${SERVER_PORT:-18765}"; \
		echo "🚀 Starting complete Neural Junkie system..."; \
		echo "   (Specialist agents are started in-process by the server via config)"; \
		if [ -f "$(SLACK_VENDOR_JSON)" ]; then echo "   Slack: bundled NJ app (vendor/oauth.json → Connect Slack enabled)"; fi; \
		go run $(SERVER_GO_TAGS) ./cmd/server & \
		echo "⏳ Waiting for hub at http://localhost:$${PORT}/api/health ..."; \
		ok=0; for i in $$(seq 1 60); do \
			if curl -sf "http://localhost:$${PORT}/api/health" | grep -q "\"status\":\"ok\""; then ok=1; break; fi; \
			sleep 1; \
		done; \
		if [ "$$ok" != "1" ]; then \
			echo "❌ Hub did not become healthy within 60s."; \
			echo "   Common cause: port $${PORT} already in use. Check: lsof -i :$${PORT}"; \
			echo "   Start the hub alone to see the error: make server"; \
			exit 1; \
		fi; \
		echo "✅ Hub is up. Opening GUI..."; \
		if [ ! -x desktop/node_modules/.bin/tauri ]; then \
			echo "📦 Desktop deps missing — installing..."; \
			$(MAKE) gui-install; \
		fi; \
		$(MAKE) ensure-sidecar ensure-ollama-bundle; \
		cd desktop && npm run tauri:dev'

demo: ## Run a complete demo
	@echo "🎬 Starting demo..."
	@echo "This will start the server and agents, then send demo messages"
	@./scripts/demo.sh

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	@rm -rf bin/
	@rm -rf *.app
	@echo "✅ Clean complete!"

cleanup-test-artifacts: ## Remove leaked test repo caches (widget-expert, temp paths) from ~/.neural-junkie
	@chmod +x ./scripts/cleanup-test-artifacts.py
	@./scripts/cleanup-test-artifacts.py --hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"

cleanup-test-artifacts-dry: ## Preview test artifact cleanup without deleting
	@chmod +x ./scripts/cleanup-test-artifacts.py
	@./scripts/cleanup-test-artifacts.py --dry-run --hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"

test: test-go ## Run Go unit tests (alias for test-go)

deps: ## Download dependencies (LoRA stack: make deps-lora when Specialist tuning pack is enabled)
	@echo "📦 Downloading dependencies..."
	@go mod download
	@echo "✅ Dependencies downloaded!"

deps-lora: ## Install LoRA training Python stack (.venv-lora)
	@chmod +x ./scripts/setup-lora-deps.sh
	@./scripts/setup-lora-deps.sh

ensure-lora-deps: ## Ensure LoRA training venv exists (install once)
	@if [ ! -x .venv-lora/bin/python ]; then \
		echo "📦 LoRA training deps missing — installing (.venv-lora)..."; \
		$(MAKE) deps-lora; \
	else \
		echo "✅ LoRA training deps ready (.venv-lora)"; \
	fi

pull-models: ## Pull required Ollama models (code tier + utility tier + LoRA bases)
	@echo "📥 Pulling Ollama models..."
	@echo "  Code tier: qwen2.5-coder:14b (~9GB)..."
	@ollama pull qwen2.5-coder:14b
	@echo "  Code tier (large): qwen3.5:27b (~17GB)..."
	@ollama pull qwen3.5:27b
	@echo "  Utility tier: qwen3.5:9b (~6.6GB)..."
	@ollama pull qwen3.5:9b
	@echo "  LoRA base: llama3:8b (~4.7GB, biology pack)..."
	@ollama pull llama3:8b
	@echo "  LoRA training base: llama3.1:8b (~4.9GB, recommended for train → compose)..."
	@ollama pull llama3.1:8b
	@echo "  LoRA bootstrap: llama3.2:3b (~2GB, security preset)..."
	@ollama pull llama3.2:3b
	@echo "  LoRA bootstrap: mistral:7b (~4.4GB, SQL preset)..."
	@ollama pull mistral:7b
	@echo "✅ All models pulled!"
	@echo ""
	@ollama list

install: build ## Install binaries to PATH
	@echo "📦 Installing binaries..."
	@mkdir -p ~/bin
	@cp bin/* ~/bin/
	@echo "✅ Installed to ~/bin/"
	@echo "   Make sure ~/bin is in your PATH"

# Repository Expert Agents
.PHONY: repo-agent demo-repo-agent setup-env

setup-env: ## Setup environment variables from env.local
	@echo "🔧 Setting up environment..."
	@if [ -f env.local ]; then \
		echo "✅ Found env.local"; \
	else \
		echo "⚠️  env.local not found, copying from env.example..."; \
		cp env.example env.local; \
		echo "📝 Please edit env.local with your AI Hub credentials"; \
	fi

repo-agent: setup-env ## Create a repository expert agent (usage: make repo-agent PATH=/path/to/repo NAME="Agent Name")
	@if [ -z "$(PATH)" ]; then \
		echo "❌ Error: PATH is required"; \
		echo "Usage: make repo-agent PATH=/path/to/repo NAME=\"Agent Name\""; \
		exit 1; \
	fi
	@source load-env.sh && \
		go run cmd/agent/main.go \
			--type repo \
			--repo-path "$(PATH)" \
			--name "$(NAME)" \
			--mock=false \
			--channel general

demo-repo-agent: setup-env ## Run repository agent demo (usage: make demo-repo-agent PATH=/path/to/repo)
	@if [ -z "$(PATH)" ]; then \
		echo "Usage: make demo-repo-agent PATH=/path/to/repo"; \
		echo "Example: make demo-repo-agent PATH=~/projects/my-app"; \
		exit 1; \
	fi
	@./scripts/demo-repo-agent.sh "$(PATH)"

# ── Cross-compile & Bundle ───────────────────────────────────────────

SIDECAR_DIR := desktop/src-tauri/binaries

.PHONY: fetch-ollama build-server-mac-arm build-server-mac-intel build-server-linux bundle-mac bundle-linux bundle release

fetch-ollama: ## Download Ollama runtime for current platform (bundled in installers)
	@chmod +x scripts/fetch-ollama.sh
	@./scripts/fetch-ollama.sh

build-server-mac-arm: ## Cross-compile server for macOS Apple Silicon
	@echo "🔨 Building server for macOS arm64..."
	@mkdir -p $(SIDECAR_DIR)
	@GOOS=darwin GOARCH=arm64 go build $(SERVER_GO_TAGS) -o $(SIDECAR_DIR)/nj-server-aarch64-apple-darwin ./cmd/server

build-server-mac-intel: ## Cross-compile server for macOS Intel
	@echo "🔨 Building server for macOS amd64..."
	@mkdir -p $(SIDECAR_DIR)
	@GOOS=darwin GOARCH=amd64 go build $(SERVER_GO_TAGS) -o $(SIDECAR_DIR)/nj-server-x86_64-apple-darwin ./cmd/server

build-server-linux: ## Cross-compile server for Linux x86_64
	@echo "🔨 Building server for Linux amd64..."
	@mkdir -p $(SIDECAR_DIR)
	@GOOS=linux GOARCH=amd64 go build $(SERVER_GO_TAGS) -o $(SIDECAR_DIR)/nj-server-x86_64-unknown-linux-gnu ./cmd/server

build-sidecar: ## Build server sidecar for current platform
	@echo "🔨 Building server sidecar for current platform... $(if $(SERVER_GO_TAGS),[Slack vendor],)"
	@mkdir -p $(SIDECAR_DIR)
	@go build $(SERVER_GO_TAGS) -o $(SIDECAR_DIR)/nj-server-$$(rustc -vV | grep host | cut -d' ' -f2) ./cmd/server

build-nj-remote: ## Build nj-remote workspace sidecar (IDE v4 SSH/devcontainer)
	@echo "🔨 Building nj-remote..."
	@mkdir -p bin
	@go build -o bin/nj-remote ./cmd/nj-remote

nj-remote-install: build-nj-remote ## Install nj-remote on remote host (TARGET=user@host, optional ROOT=, PORT=)
	@if [ -z "$(TARGET)" ]; then \
		echo "❌ TARGET is required (e.g. make nj-remote-install TARGET=ec2-user@host)"; \
		exit 1; \
	fi
	@./scripts/install-nj-remote.sh --remote "$(TARGET)" $(if $(ROOT),--root "$(ROOT)",) $(if $(PORT),--port "$(PORT)",)

bundle-mac: build-server-mac-arm fetch-ollama ## Build production desktop app for macOS
	@echo "📦 Building macOS bundle..."
	@cd desktop && npm run tauri:build
	@echo "✅ macOS bundle ready at desktop/src-tauri/target/release/bundle/"

bundle-linux: build-server-linux fetch-ollama ## Build production desktop app for Linux
	@echo "📦 Building Linux bundle..."
	@cd desktop && npm run tauri:build
	@echo "✅ Linux bundle ready at desktop/src-tauri/target/release/bundle/"

bundle: ## Build bundles for current platform
	@$(MAKE) build-sidecar
	@$(MAKE) fetch-ollama
	@cd desktop && npm run tauri:build

release: ## Tag and push a release (usage: make release VERSION=1.2.0)
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ Error: VERSION is required"; \
		echo "Usage: make release VERSION=1.2.0"; \
		exit 1; \
	fi
	@echo "🏷️  Releasing v$(VERSION)..."
	@echo "   (tauri.conf.json package.version is set by CI from the tag — see docs/RELEASE_UPDATES.md)"
	@cd desktop && sed -i.bak 's/^version = "[^"]*"/version = "$(VERSION)"/' src-tauri/Cargo.toml && rm -f src-tauri/Cargo.toml.bak
	@cd desktop && npm version $(VERSION) --no-git-tag-version 2>/dev/null || true
	@cd desktop/src-tauri && cargo check -q 2>/dev/null || true
	@git add desktop/package.json desktop/package-lock.json desktop/src-tauri/Cargo.toml desktop/src-tauri/Cargo.lock
	@git commit -m "release: v$(VERSION)"
	@git tag v$(VERSION)
	@echo "✅ Tagged v$(VERSION). Push with: git push && git push origin v$(VERSION)"
