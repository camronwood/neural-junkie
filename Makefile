.PHONY: help build local-build local-install run-server run-agents run-all demo clean docs stop refresh test test-go test-all test-messages slack-vendor-check slack-vendor-json gallery-sync articles-sync site-nav-sync site-seo-sync github-metadata-sync deps-lora server-regression server-debug collab-scenarios-all collab-scenarios-core collab-preflight slack-smoke release-help test-regression-live chat-scenarios-debug test-parity-stable test-parity-stable-restart test-parity-full-restart parity-scenarios parity-scenarios-list test-regression-bundle test-conversation-contract test-transcript-metrics test-everything test-everything-full release-prep release-prep-fix-loop bump-homebrew-cask layer-gate layer-fix-loop layer-list layer-climb layer-overnight overnight overnight-preflight overnight-release-prep overnight-release-prep-fix-loop ensure-ollama-models-ready slack-oauth-relay-deploy-cf slack-oauth-relay-deploy test-growth-loop test-growth-once test-growth-list check-catalog-downloads sync-sd-pack

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
	@echo "Release/testing workflow: make release-help"
	@echo "Documentation: make docs"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z0-9_.-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		grep -Ev '^(overnight-release-prep|overnight-release-prep-fix-loop|test-regression-live|test-regression-bundle|chat-scenarios-regression|conversation-scenarios-regression):' | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-22s %s\n", $$1, $$2}'

release-help: ## Release & testing workflow — start here (layers, overnight, full gate)
	@echo "Neural Junkie — release & testing commands"
	@echo "=========================================="
	@echo ""
	@echo "PRIMARY (use these day-to-day — each boots Ollama + hub automatically)"
	@echo "  make layer-list                         # layers in order + time estimates"
	@echo "  make layer-gate LAYER=implement         # test ONE layer (~15m–3h)"
	@echo "  make layer-fix-loop LAYER=chat          # layer test → Cursor fix → verify"
	@echo "  make test-growth-loop                   # discover gaps → add/strengthen tests"
	@echo "  make layer-climb                        # run layers until first failure"
	@echo "  make layer-climb CONTINUE=1             # run all layers; rollup + live status file"
	@echo "  #   progress: docs/testing/layer-climb-status.txt  (tail -f)"
	@echo "  make model-benchmark SUITE=standard     # multi-model live benchmark (boots stack)"
	@echo "  make overnight-preflight                # afternoon check before overnight"
	@echo "  make overnight                          # walk-away full release-prep (tmux)"
	@echo "  make layer-overnight LAYER=implement    # walk-away layer fix loop"
	@echo ""
	@echo "FULL GATE (only after layers pass)"
	@echo "  make release-prep                       # test-everything-full + parity + benchmark"
	@echo "  make release-prep-fix-loop              # full gate + Cursor agent fix loop"
	@echo "  make overnight NJ_OVERNIGHT_TARGET=release-prep-fix-loop"
	@echo ""
	@echo "CI (fast, no hub)"
	@echo "  make test-all                           # vet + Go + desktop tsc + Vitest"
	@echo "  make test-conversation-contract         # agent/hub/desktop wiring"
	@echo "  make test-scenario-assert               # scenario contract unit tests"
	@echo ""
	@echo "LAYERS (make layer-gate LAYER=<name>)"
	@echo "  ci            test-all + conversation-contract"
	@echo "  implement     implement-scenarios (20/20)"
	@echo "  chat          chat + conversation regression"
	@echo "  collab        collab edge-case regression (~13)"
	@echo "  collab-core   participation/planning core (~8; fix-loop target)"
	@echo "  collab-full   all collab scenarios (25)"
	@echo "  bundle        implement + chat + conversation"
	@echo "  parity        3x implement with hub restart"
	@echo ""
	@echo "DEBUG (single scenario)"
	@echo "  make implement-scenario SCENARIO=go-handler"
	@echo "  make chat-scenario SCENARIO=thanks-closure"
	@echo "  make collab-scenario SCENARIO=planning-two-agent"
	@echo "  make *-scenarios-list                   # list scenario names"
	@echo ""
	@echo "  make gen-pack-capabilities             # regenerate TS capability tokens from JSON"

test-regression-live: release-help

docs: ## Show documentation guide
	@cat DOCS.md

gen-pack-capabilities: ## Regenerate desktop pack capability tokens from JSON
	@python3 scripts/gen-pack-capabilities.py

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

articles-sync: ## Regenerate docs/articles from campaigns LinkedIn sources
	@chmod +x ./scripts/sync-articles.sh
	@./scripts/sync-articles.sh
	@python3 ./scripts/sync-site-nav.py

site-nav-sync: ## Unify header navigation across all docs/*.html pages
	@python3 ./scripts/sync-site-nav.py

site-seo-sync: ## Sync nav + SEO meta (canonical, OG, Twitter) and regenerate sitemap/robots
	@python3 ./scripts/sync-site-nav.py
	@python3 ./scripts/generate-sitemap.py

github-metadata-sync: ## Update GitHub repo descriptions, homepages, and topics (requires gh auth)
	@chmod +x ./scripts/sync-github-repo-metadata.sh
	@./scripts/sync-github-repo-metadata.sh

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
	@echo "💡 Packaged desktop app: make local-build"
	@echo "   (optional install on macOS: make local-build INSTALL=1)"

sync-sd-pack: ## Dev-link real software-development pack (Mach-O sd-mcp-server) into ~/.neural-junkie
	@chmod +x scripts/sync-sd-pack-dev.sh
	@./scripts/sync-sd-pack-dev.sh

run-server: ## Start the chat hub server
	@echo "🚀 Starting chat hub server on http://localhost:18765 $(if $(SERVER_GO_TAGS),[Slack vendor],)"
	@NEURAL_JUNKIE_RELAXED_LOCAL=1 go run $(SERVER_GO_TAGS) ./cmd/server

run-hub: run-server ## Alias: dev hub with NEURAL_JUNKIE_RELAXED_LOCAL=1 (loopback synthetic member)

server: setup-env ## Start server with environment loaded
	@echo "🚀 Starting chat hub server with environment from env.local... $(if $(SERVER_GO_TAGS),[Slack vendor],)"
	@bash -c 'source load-env.sh && NEURAL_JUNKIE_RELAXED_LOCAL=1 go run $(SERVER_GO_TAGS) ./cmd/server'

server-debug: setup-env ## Hub with NEURAL_JUNKIE_DEBUG=1 (pprof + /api/debug/hub-memory); logs to /tmp/nj-hub.log
	@echo "🔧 Starting debug hub → /tmp/nj-hub.log  (pprof: http://127.0.0.1:6060/debug/pprof/) $(if $(SERVER_GO_TAGS),[Slack vendor],)"
	@bash -c 'source load-env.sh && NEURAL_JUNKIE_DEBUG=1 go run $(SERVER_GO_TAGS) ./cmd/server 2>&1 | tee /tmp/nj-hub.log'

server-regression: setup-env ## Hub for live scenario regression (RATE_LIMIT=0 + DEBUG=1); logs to /tmp/nj-hub.log
	@echo "🔧 Regression hub → /tmp/nj-hub.log  (NEURAL_JUNKIE_RATE_LIMIT=0 NJ_OLLAMA_MAX_CONCURRENCY=2 NEURAL_JUNKIE_DEBUG=1 NEURAL_JUNKIE_SLACK_DISABLED=1) $(if $(SERVER_GO_TAGS),[Slack vendor],)"
	@bash -c 'source load-env.sh && NEURAL_JUNKIE_RATE_LIMIT=0 NJ_OLLAMA_MAX_CONCURRENCY=2 NEURAL_JUNKIE_DEBUG=1 NEURAL_JUNKIE_SLACK_DISABLED=1 go run $(SERVER_GO_TAGS) ./cmd/server 2>&1 | tee /tmp/nj-hub.log'

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

slack-smoke: ## Slack integration smoke (CI: unit tests; LIVE=1: synthetic inbound live check, no Slack posts; SLACK_SMOKE_OUTBOUND=1: optional gated outbound)
	@go test ./internal/integrations/slack/... -count=1
	@go test ./cmd/server -run 'TestHandleSlack' -count=1
	@if [ "$(LIVE)" = "1" ]; then \
		chmod +x scripts/slack-live-smoke.sh; \
		./scripts/slack-live-smoke.sh; \
	fi

collab-preflight: ## Fail-fast checks before collab-scenarios-all (hub, Ollama, agents, scenario list)
	@bash -c 'source load-env.sh && NJ_REGRESSION_SLIM_ROSTER=1 python3 scripts/collab-preflight.py $(if $(REQUIRE_GEMINI),--require-gemini,)'

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
	@if [ -z "$(SCENARIO)" ]; then echo "Usage: make collab-scenario SCENARIO=planning-two-agent [PROFILE=fast] [KEEP=1] [VERBOSE=1] [REQUIRE_GEMINI=1]"; exit 1; fi
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --scenario "$(SCENARIO)" \
		$(if $(PROFILE),--profile $(PROFILE),) \
		$(if $(VERBOSE),--verbose,) \
		$(if $(KEEP),--keep,) \
		$(if $(REQUIRE_GEMINI),--require-gemini,)

collab-scenarios: ## Run all live collab scenarios (hub should use make server-regression)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --all \
		$(if $(PROFILE),--profile $(PROFILE),) \
		$(if $(VERBOSE),--verbose,) \
		$(if $(REQUIRE_GEMINI),--require-gemini,)

collab-scenarios-all: collab-scenarios ## Alias: full collab sweep (24 scenarios; PROFILE does not shorten timeouts)

collab-scenarios-core: ## Collab core participation/planning (~8 scenarios; hub restart between)
	@NEURAL_JUNKIE_RATE_LIMIT=0 NJ_RESTART_HUB_BETWEEN_SCENARIOS=1 \
		NJ_REQUIRE_FULL_BOOT=1 NJ_REGRESSION_SLIM_ROSTER=1 \
		NJ_SCENARIO_PROFILE=core python3 scripts/collab-scenarios.py --core \
		$(if $(PROFILE),--profile $(PROFILE),--profile core) \
		$(if $(VERBOSE),--verbose,) \
		$(if $(KEEP),--keep,)

collab-scenarios-core-debug: ## Serial collab-core repro (one scenario at a time; stop on FAIL)
	@chmod +x scripts/collab-sweep-serial.sh
	@CORE=1 RETRIES=$(or $(RETRIES),1) RESUME=$(RESUME) VERBOSE=$(VERBOSE) \
		NJ_REQUIRE_FULL_BOOT=1 NJ_REGRESSION_SLIM_ROSTER=1 \
		NJ_SCENARIO_PROFILE=core ./scripts/collab-sweep-serial.sh

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

runbook-scenario: ## Run one runbook scenario JSON (SCENARIO=health-check-branch; hub must be running)
	@if [ -z "$(SCENARIO)" ]; then echo "Usage: make runbook-scenario SCENARIO=health-check-branch"; exit 1; fi
	@chmod +x scripts/runbook-scenarios.py
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/runbook-scenarios.py --scenario "$(SCENARIO)" $(if $(VERBOSE),--verbose,)
	@go test ./test/ -run TestRunbookScenario_$(SCENARIO) -count=1

collab-routing-matrix: ## A/B smart routing on execute-deliverable (needs live hub + agents)
	@chmod +x scripts/collab-routing-matrix.sh
	@NEURAL_JUNKIE_RATE_LIMIT=0 ./scripts/collab-routing-matrix.sh

collab-parity: ## Solo vs collab deliverable parity on minimal-repo fixture
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py --scenario solo-vs-collab-parity \
		$(if $(PROFILE),--profile $(PROFILE),--profile fast,) \
		$(if $(VERBOSE),--verbose,)

collab-scenario-regression: ## Collab edge-case regression (~12 scenarios, one process; prefer: make layer-gate LAYER=collab)
	@NEURAL_JUNKIE_RATE_LIMIT=0 NJ_REGRESSION_SLIM_ROSTER=1 NJ_REGRESSION_COLLAB_EDGE=1 \
		NJ_OLLAMA_MAX_CONCURRENCY=2 NJ_REQUIRE_FULL_BOOT=1 \
		python3 scripts/collab-scenarios.py --suite edge $(if $(VERBOSE),--verbose,)

conversation-scenarios-regression:
	@echo "Use: make layer-gate LAYER=chat  (chat + conversation regression are the chat layer)" >&2
	@exit 1

test-collab-plan: ## Deterministic Go tests for collab plan parsing regressions (CI-safe)
	@go test ./internal/collaboration/... ./internal/hub/... -count=1 -run 'Regression|DependencyProse|Findings|4ea36409|f7518f88|DocumentFindings|DistinctDeliverable|StackTool|FilterCollab|SuppressMCP'

test-conversation-contract: ## CI-safe conversation + collab wiring contract (agent, hub, desktop, smoke)
	@echo "🧪 Deterministic transcript metrics..."
	@$(MAKE) test-transcript-metrics
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
	@cd scripts/lib && PYTHONPATH=.. python3 -m unittest scenario_assert_test.py scenario_contract_test.py transcript_contract_test.py test_growth_candidates_test.py test_growth_guardrails_test.py collab_hub_test.py hub_regression_test.py hub_auth_test.py release_prep_failures_test.py fix_loop_git_test.py hub_cleanup_test.py scenario_flake_retry_test.py release_prep_layers_test.py regression_boot_test.py regression_models_test.py regression_collab_test.py
	@PYTHONPATH=scripts python3 scripts/lib/scenario_contract.py

test-transcript-metrics: ## Deterministic sanitized conversation metrics (no hub, judge, retry, or network)
	@NJ_SCENARIO_FLAKE_RETRY=0 PYTHONHASHSEED=0 PYTHONPATH=scripts python3 scripts/transcript-metrics.py

check-catalog-downloads: ## Opt-in network check: each packs/catalog.json download_url returns HTTP 200
	@python3 scripts/check-catalog-downloads.py

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

chat-scenarios-regression:
	@echo "Use: make layer-gate LAYER=chat  (chat + conversation regression are the chat layer)" >&2
	@exit 1

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
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/implement-scenarios-stable.py --runs 3 --min-pass 20 \
		--restart-between --hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"

test-parity-stable-stress: ## Run implement-scenarios 3x back-to-back (may OOM hub on tight memory)
	@NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/implement-scenarios-stable.py --runs 3 --min-pass 20 \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"

test-parity-stable-restart: test-parity-stable ## Alias for stable gate (restart between sweeps)

test-regression-bundle:
	@$(MAKE) layer-gate LAYER=bundle VERBOSE=$(VERBOSE) NO_RESTART_HUB=$(NO_RESTART_HUB)

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

overnight-preflight: ## Afternoon check before overnight (models + hub + Arena + release-prep-ready)
	@chmod +x scripts/overnight-preflight.sh scripts/ensure-ollama-models-ready.py
	@BENCHMARK_SUITE='$(or $(BENCHMARK_SUITE),release)' \
	 NEURAL_JUNKIE_HUB_URL='$(NEURAL_JUNKIE_HUB_URL)' \
	 NJ_OVERNIGHT_KEEP_ALIVE='$(or $(NJ_OVERNIGHT_KEEP_ALIVE),24h)' \
	 ./scripts/overnight-preflight.sh

overnight: ## Walk-away clean gate: reset + hub + preflight + release-prep in tmux (see scripts/overnight.sh)
	@chmod +x scripts/overnight.sh scripts/ensure-ollama-models-ready.py
	@NJ_OVERNIGHT_TARGET='$(or $(NJ_OVERNIGHT_TARGET),release-prep)' \
	 BENCHMARK_SUITE='$(or $(BENCHMARK_SUITE),release)' \
	 NO_PULL='$(or $(NO_PULL),1)' \
	 SKIP_LIVE='$(SKIP_LIVE)' SKIP_PARITY='$(SKIP_PARITY)' SKIP_BENCHMARK='$(SKIP_BENCHMARK)' \
	 NO_FULL='$(NO_FULL)' SKIP_EVERYTHING='$(SKIP_EVERYTHING)' \
	 BENCHMARK_MODELS='$(BENCHMARK_MODELS)' BENCHMARK_ALLOW_LARGE='$(BENCHMARK_ALLOW_LARGE)' \
	 NO_RESTART_HUB='$(NO_RESTART_HUB)' VERBOSE='$(VERBOSE)' STOP_ON_FAIL='$(STOP_ON_FAIL)' \
	 PULL='$(PULL)' NJ_OVERNIGHT_KEEP_ALIVE='$(NJ_OVERNIGHT_KEEP_ALIVE)' \
	 NEURAL_JUNKIE_HUB_URL='$(NEURAL_JUNKIE_HUB_URL)' IN_TMUX='$(IN_TMUX)' \
	 NJ_OVERNIGHT_SESSION='$(NJ_OVERNIGHT_SESSION)' NJ_OVERNIGHT_LOG='$(NJ_OVERNIGHT_LOG)' \
	 MAX_ITER='$(MAX_ITER)' REPORT='$(REPORT)' SKIP_RELEASE_PREP='$(SKIP_RELEASE_PREP)' \
	 SKIP_AGENT='$(SKIP_AGENT)' SKIP_VERIFY='$(SKIP_VERIFY)' DRY_RUN='$(DRY_RUN)' \
	 MODEL='$(MODEL)' PREFER_SDK='$(PREFER_SDK)' AGENT_TIMEOUT='$(AGENT_TIMEOUT)' \
	 NO_COMMIT='$(NO_COMMIT)' FIX_BRANCH='$(FIX_BRANCH)' BASE_BRANCH='$(BASE_BRANCH)' \
	 LAYER='$(LAYER)' SKIP_GATE='$(SKIP_GATE)' \
	 ./scripts/overnight.sh

layer-list: ## List release-prep layers in recommended order (ci → implement → … → parity)
	@chmod +x scripts/layer-gate.py
	@python3 scripts/layer-gate.py --layer ci --list

layer-climb: ## Run layers in order (default stop on fail; CONTINUE=1 runs all + writes docs/testing/layer-climb-*.md)
	@chmod +x scripts/layer-climb.py scripts/layer-gate.py
	@bash -c 'source load-env.sh && NEURAL_JUNKIE_RATE_LIMIT=0 \
		python3 scripts/layer-climb.py \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}" \
		$(if $(CONTINUE),--continue-on-fail,) \
		$(if $(VERBOSE),--verbose,) \
		$(if $(NO_RESTART_HUB),--no-restart-hub,)'

layer-gate: ## Run one layer gate (LAYER=ci|implement|chat|collab|collab-core|collab-full|bundle|parity)
	@if [ -z "$(LAYER)" ]; then echo "Usage: make layer-gate LAYER=implement [VERBOSE=1] [NO_RESTART_HUB=1]"; $(MAKE) layer-list; exit 1; fi
	@chmod +x scripts/layer-gate.py
	@bash -c 'source load-env.sh && NEURAL_JUNKIE_RATE_LIMIT=0 \
		$(if $(filter collab collab-core,$(LAYER)),NJ_REQUIRE_FULL_BOOT=1 NJ_REGRESSION_SLIM_ROSTER=1,) \
		python3 scripts/layer-gate.py \
		--layer "$(LAYER)" \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}" \
		$(if $(VERBOSE),--verbose,) \
		$(if $(NO_RESTART_HUB),--no-restart-hub,)'

layer-fix-loop: ## Layer gate + Cursor agent fix loop (LAYER=implement MAX_ITER=3 DRY_RUN=1 NO_WORKTREE=1)
	@if [ -z "$(LAYER)" ]; then echo "Usage: make layer-fix-loop LAYER=implement [MAX_ITER=3] [DRY_RUN=1] [NO_COMMIT=1] [NO_WORKTREE=1]"; $(MAKE) layer-list; exit 1; fi
	@chmod +x scripts/layer-fix-loop.py scripts/layer-gate.py
	@bash -c 'source load-env.sh && NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/layer-fix-loop.py \
		--layer "$(LAYER)" \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}" \
		--max-iterations $${MAX_ITER:-3} \
		$(if $(REPORT),--report "$(REPORT)",) \
		$(if $(SKIP_GATE),--skip-gate,) \
		$(if $(SKIP_AGENT),--skip-agent,) \
		$(if $(SKIP_VERIFY),--skip-verify,) \
		$(if $(DRY_RUN),--dry-run,) \
		$(if $(NO_RESTART_HUB),--no-restart-hub,) \
		$(if $(VERBOSE),--verbose,) \
		$(if $(MODEL),--model "$(MODEL)",) \
		$(if $(PREFER_SDK),--prefer-sdk,) \
		$(if $(AGENT_TIMEOUT),--agent-timeout $(AGENT_TIMEOUT),) \
		$(if $(NO_COMMIT),--no-commit,) \
		$(if $(FIX_BRANCH),--fix-branch "$(FIX_BRANCH)",) \
		$(if $(BASE_BRANCH),--base-branch "$(BASE_BRANCH)",) \
		$(if $(NO_WORKTREE),--no-worktree,--use-worktree)'

layer-overnight: ## Walk-away layer fix loop in tmux (LAYER=implement)
	@if [ -z "$(LAYER)" ]; then echo "Usage: make layer-overnight LAYER=implement"; $(MAKE) layer-list; exit 1; fi
	@$(MAKE) overnight NJ_OVERNIGHT_TARGET=layer-fix-loop LAYER='$(LAYER)' \
	 MAX_ITER='$(MAX_ITER)' NO_COMMIT='$(NO_COMMIT)' AGENT_TIMEOUT='$(AGENT_TIMEOUT)' \
	 REPORT='$(REPORT)' SKIP_GATE='$(SKIP_GATE)' SKIP_AGENT='$(SKIP_AGENT)' \
	 SKIP_VERIFY='$(SKIP_VERIFY)' DRY_RUN='$(DRY_RUN)' MODEL='$(MODEL)' \
	 PREFER_SDK='$(PREFER_SDK)' FIX_BRANCH='$(FIX_BRANCH)' BASE_BRANCH='$(BASE_BRANCH)' \
	 NO_WORKTREE='$(NO_WORKTREE)' VERBOSE='$(VERBOSE)' IN_TMUX='$(IN_TMUX)'

test-growth-list: ## List ranked test-growth candidates (no agent)
	@chmod +x scripts/test-growth-loop.py
	@python3 scripts/test-growth-loop.py --list

test-growth-once: ## Single test-growth iteration (MAX_ITER=1)
	@$(MAKE) test-growth-loop MAX_ITER=1 $(if $(DRY_RUN),DRY_RUN=1,) $(if $(NO_COMMIT),NO_COMMIT=1,) $(if $(SKIP_LIVE),SKIP_LIVE=1,) $(if $(SKIP_AGENT),SKIP_AGENT=1,)

test-growth-loop: ## Test-growth loop: discover gaps → Cursor agent → verify → commit (MAX_ITER=3)
	@chmod +x scripts/test-growth-loop.py
	@bash -c 'source load-env.sh && python3 scripts/test-growth-loop.py \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}" \
		--max-iterations $${MAX_ITER:-3} \
		$(if $(CANDIDATE_KIND),--candidate-kind "$(CANDIDATE_KIND)",) \
		$(if $(SKIP_AGENT),--skip-agent,) \
		$(if $(SKIP_VERIFY),--skip-verify,) \
		$(if $(SKIP_LIVE),--skip-live,) \
		$(if $(DRY_RUN),--dry-run,) \
		$(if $(VERBOSE),--verbose,) \
		$(if $(MODEL),--model "$(MODEL)",) \
		$(if $(PREFER_SDK),--prefer-sdk,) \
		$(if $(AGENT_TIMEOUT),--agent-timeout $(AGENT_TIMEOUT),) \
		$(if $(NO_COMMIT),--no-commit,) \
		$(if $(GROWTH_BRANCH),--growth-branch "$(GROWTH_BRANCH)",) \
		$(if $(BASE_BRANCH),--base-branch "$(BASE_BRANCH)",) \
		$(if $(USE_WORKTREE),--use-worktree,--no-worktree) \
		$(if $(STABILITY_RUNS),--stability-runs $(STABILITY_RUNS),)'

release-prep-fix-loop: ## Release gate + Cursor agent fix loop (REPORT=path DRY_RUN=1 SKIP_BENCHMARK=1 MAX_ITER=3 NO_COMMIT=1)
	@chmod +x scripts/release-prep-fix-loop.py scripts/release-prep.py
	@bash -c 'source load-env.sh && NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/release-prep-fix-loop.py \
		--hub "$${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}" \
		--max-iterations $${MAX_ITER:-3} \
		$(if $(REPORT),--report "$(REPORT)",) \
		$(if $(SKIP_RELEASE_PREP),--skip-release-prep,) \
		$(if $(SKIP_AGENT),--skip-agent,) \
		$(if $(SKIP_VERIFY),--skip-verify,) \
		$(if $(DRY_RUN),--dry-run,) \
		$(if $(SKIP_BENCHMARK),--skip-benchmark,) \
		$(if $(NO_FULL),--no-full,) \
		$(if $(SKIP_PARITY),--skip-parity,) \
		$(if $(SKIP_LIVE),--skip-live,) \
		$(if $(VERBOSE),--verbose,) \
		$(if $(MODEL),--model "$(MODEL)",) \
		$(if $(PREFER_SDK),--prefer-sdk,) \
		$(if $(AGENT_TIMEOUT),--agent-timeout $(AGENT_TIMEOUT),) \
		$(if $(NO_COMMIT),--no-commit,) \
		$(if $(FIX_BRANCH),--fix-branch "$(FIX_BRANCH)",) \
		$(if $(BASE_BRANCH),--base-branch "$(BASE_BRANCH)",) \
		$(if $(USE_WORKTREE),--use-worktree,--no-worktree)'

overnight-release-prep-fix-loop:
	@$(MAKE) overnight NJ_OVERNIGHT_TARGET=release-prep-fix-loop \
	 MAX_ITER='$(MAX_ITER)' REPORT='$(REPORT)' SKIP_RELEASE_PREP='$(SKIP_RELEASE_PREP)' \
	 SKIP_AGENT='$(SKIP_AGENT)' SKIP_VERIFY='$(SKIP_VERIFY)' DRY_RUN='$(DRY_RUN)' \
	 MODEL='$(MODEL)' PREFER_SDK='$(PREFER_SDK)' AGENT_TIMEOUT='$(AGENT_TIMEOUT)' \
	 NO_COMMIT='$(NO_COMMIT)' FIX_BRANCH='$(FIX_BRANCH)' BASE_BRANCH='$(BASE_BRANCH)' \
	 USE_WORKTREE='$(USE_WORKTREE)' \
	 SKIP_LIVE='$(SKIP_LIVE)' SKIP_PARITY='$(SKIP_PARITY)' SKIP_BENCHMARK='$(SKIP_BENCHMARK)' \
	 NO_FULL='$(NO_FULL)' VERBOSE='$(VERBOSE)' IN_TMUX='$(IN_TMUX)'

release-prep: ## Full release gate: test-everything-full + parity-restart + release benchmark (2 models) → docs/testing/release-prep-*.md
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

overnight-release-prep: overnight

bump-homebrew-cask: bump-homebrew-tap ## Regenerate homebrew tap (cask + Linux formula)

bump-homebrew-tap: ## Regenerate ../homebrew-tap cask + Linux formula (TAG=v1.2.0-beta.5 TAP_DIR=../homebrew-tap)
	@chmod +x scripts/bump-homebrew-tap.sh scripts/bump-homebrew-cask.sh
	@./scripts/bump-homebrew-tap.sh '$(or $(TAG),v1.2.0-beta.5)' '$(or $(TAP_DIR),$(CURDIR)/../homebrew-tap)'

ensure-ollama-models-ready: ## Pull/warm/smoke Ollama models before release prep (SUITE=release; NO_PULL=1 to skip pulls)
	@chmod +x scripts/ensure-ollama-models-ready.py
	@python3 scripts/ensure-ollama-models-ready.py \
		$(if $(NO_PULL),,--pull-missing) --warm --smoke \
		--keep-alive $(or $(NJ_OVERNIGHT_KEEP_ALIVE),24h) \
		--suite $(or $(SUITE),quick) \
		$(if $(SKIP_BENCHMARK),--skip-benchmark,) \
		$(if $(BENCHMARK_ALLOW_LARGE),--allow-large-models,) \
		$(if $(MODELS),--models "$(MODELS)",)

model-benchmark: ## Auto-boots Ollama+hub; benchmark coder models ≤~9 GB (SUITE=quick; SKIP_BOOT=1 to reuse; NO_PULL=1; BENCHMARK_ALLOW_LARGE=1)
	@chmod +x scripts/model-benchmark-suite.py
	@BENCHMARK_SUITE=$(or $(SUITE),quick) \
		$(if $(NO_PULL),NO_PULL=1,) \
		$(if $(MODELS),BENCHMARK_MODELS="$(MODELS)",) \
		$(if $(BENCHMARK_ALLOW_LARGE),BENCHMARK_ALLOW_LARGE=1,) \
		NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/model-benchmark-suite.py \
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
	@lsof -ti :1420 2>/dev/null | xargs kill -9 2>/dev/null || true
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

gui-build: ## Build production desktop app (Tauri only; prefer make local-build)
	@echo "🔨 Building desktop app..."
	@cd desktop && npm run tauri:build
	@echo "✅ Desktop app built! Check desktop/src-tauri/target/release/bundle/"

# Fresh sidecar + Tauri package for local soak testing (not a GitHub release).
# Usage:
#   make local-build              # write .app/.dmg under desktop/src-tauri/target/release/bundle/
#   make local-build INSTALL=1    # also install to /Applications on macOS
#   make local-install            # install an already-built .app
local-build: ensure-lora-deps ensure-ollama-bundle build-sidecar ## Packaged local desktop app (sidecar + Tauri); INSTALL=1 → /Applications
	@echo "🔨 Building local packaged desktop app..."
	@set -e; \
	KEY="$$HOME/.tauri/neural-junkie.key"; \
	if [ -f "$$KEY" ]; then \
		export TAURI_SIGNING_PRIVATE_KEY="$$(cat "$$KEY")"; \
		export TAURI_SIGNING_PRIVATE_KEY_PASSWORD="$${TAURI_SIGNING_PRIVATE_KEY_PASSWORD:-}"; \
		echo "   Updater signing key: $$KEY"; \
	else \
		echo "   ⚠️  Missing $$KEY — updater signing may fail (bundles may still be produced)"; \
	fi; \
	export CARGO_TARGET_DIR="$(CURDIR)/desktop/src-tauri/target"; \
	cd desktop && npm run tauri:build; \
	BUNDLE="$(CURDIR)/desktop/src-tauri/target/release/bundle"; \
	echo "✅ Local build ready under $$BUNDLE"; \
	ls -ld "$$BUNDLE/macos/"*.app 2>/dev/null || true; \
	ls -ld "$$BUNDLE/dmg/"*.dmg 2>/dev/null || true; \
	ls -ld "$$BUNDLE/deb/"*.deb 2>/dev/null || true; \
	ls -ld "$$BUNDLE/msi/"* 2>/dev/null || true; \
	if [ "$(INSTALL)" = "1" ]; then $(MAKE) local-install; fi

local-install: ## Install latest local macOS .app to /Applications (after make local-build)
	@APP="$(CURDIR)/desktop/src-tauri/target/release/bundle/macos/Neural Junkie.app"; \
	if [ ! -d "$$APP" ]; then \
		echo "❌ No app at $$APP"; \
		echo "   Run: make local-build"; \
		exit 1; \
	fi; \
	echo "📦 Installing to /Applications/Neural Junkie.app ..."; \
	osascript -e 'quit app "Neural Junkie"' 2>/dev/null || true; \
	sleep 1; \
	rm -rf "/Applications/Neural Junkie.app"; \
	cp -R "$$APP" /Applications/; \
	xattr -cr "/Applications/Neural Junkie.app" 2>/dev/null || true; \
	VER=$$(defaults read "/Applications/Neural Junkie.app/Contents/Info.plist" CFBundleShortVersionString 2>/dev/null || echo unknown); \
	echo "✅ Installed version $$VER"; \
	echo "   open \"/Applications/Neural Junkie.app\""

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
		NEURAL_JUNKIE_RELAXED_LOCAL=1 go run $(SERVER_GO_TAGS) ./cmd/server & \
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

test: test-all ## Run full CI suite (prefer test-all; test-go is Go-only)

deps: ## Download dependencies (LoRA stack: make deps-lora when Specialist tuning pack is enabled)
	@echo "📦 Downloading dependencies..."
	@go mod download
	@echo "✅ Dependencies downloaded!"

deps-lora: ## Install LoRA training Python stack (.venv-lora)
	@chmod +x ./scripts/setup-lora-deps.sh
	@./scripts/setup-lora-deps.sh

deps-lora-mlx: ## Install MLX LoRA training stack (.venv-lora-mlx, Apple Silicon)
	@python3 -m venv .venv-lora-mlx
	@.venv-lora-mlx/bin/pip install -q -r requirements-lora-mlx.txt

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
