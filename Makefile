.PHONY: help build run-server run-agents run-all demo clean docs stop refresh

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

build: ## Build all binaries
	@echo "🔨 Building server..."
	@go build -o bin/server cmd/server/main.go
	@echo "🔨 Building agent runner..."
	@go build -o bin/agent cmd/agent/main.go
	@echo "🔨 Building helper agent runner..."
	@go build -o bin/helper-agent cmd/helper-agent/main.go
	@echo "🔨 Building CLI..."
	@go build -o bin/cli cmd/cli/main.go
	@echo "🔨 Building interactive chat..."
	@go build -o bin/chat cmd/chat/main.go
	@echo "✅ Build complete!"
	@echo ""
	@echo "💡 For GUI, run: make gui-build"

run-server: ## Start the chat hub server
	@echo "🚀 Starting chat hub server on http://localhost:8080"
	@go run cmd/server/main.go

server: setup-env ## Start server with environment loaded
	@echo "🚀 Starting chat hub server with environment from env.local..."
	@bash -c 'source load-env.sh && go run cmd/server/main.go'

chat: ## Start interactive chat client
	@echo "💬 Starting interactive chat client..."
	@go run cmd/chat/main.go

gui: ## Start GUI desktop app (Tauri + React)
	@echo "🖥️  Starting desktop app with React..."
	@cd desktop && npm run tauri:dev

gui-install: ## Install GUI dependencies (first time only)
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

test-all: ## Run all tests
	@echo "🧪 Running Go unit tests..."
	@go test ./...
	@echo ""
	@echo "✅ All tests complete!"

demo-messages: ## Send demo messages to test the system
	@./scripts/demo-messages.sh

run-agents: ## Start all agent types
	@echo "🤖 Starting agents..."
	@go run cmd/agent/main.go --type frontend --name "React Expert" --channel general &
	@sleep 2
	@go run cmd/agent/main.go --type backend --name "Go Master" --channel general &
	@sleep 2
	@go run cmd/agent/main.go --type devops --name "Cloud Architect" --channel general &
	@sleep 2
	@go run cmd/agent/main.go --type database --name "SQL Expert" --channel general &
	@sleep 2
	@go run cmd/agent/main.go --type security --name "InfoSec Pro" --channel general &
	@echo "✅ All agents started!"

# Individual agent targets with environment loaded
agent-backend: setup-env ## Start Go Expert backend agent
	@echo "🤖 Starting Go Expert (Backend)..."
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type backend --name "Go Expert"'

agent-frontend: setup-env ## Start React Expert frontend agent
	@echo "🤖 Starting React Expert (Frontend)..."
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type frontend --name "React Expert"'

agent-database: setup-env ## Start SQL Master database agent
	@echo "🤖 Starting SQL Master (Database)..."
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type database --name "SQL Master"'

agent-security: setup-env ## Start Security Expert agent
	@echo "🤖 Starting Security Expert..."
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type security --name "Security Expert"'

agent-devops: setup-env ## Start DevOps Pro agent
	@echo "🤖 Starting DevOps Pro..."
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type devops --name "DevOps Pro"'

helper-agent: setup-env ## Start a helper agent (usage: make helper-agent NAME=day-one CHANNEL=general)
	@if [ -z "$(NAME)" ]; then \
		echo "❌ Error: NAME is required"; \
		echo "Usage: make helper-agent NAME=<helper-name> CHANNEL=<channel>"; \
		echo "Example: make helper-agent NAME=day-one CHANNEL=general"; \
		exit 1; \
	fi
	@echo "🎯 Starting helper agent: $(NAME)..."
	@bash -c 'source load-env.sh && go run cmd/helper-agent/main.go --name "$(NAME)" --channel "$${CHANNEL:-general}"'

agents: setup-env ## Start all agents with environment loaded
	@echo "🤖 Starting all agents with environment from env.local..."
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type backend --name "GoExpert" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 2
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type database --name "SQLMaster" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type security --name "SecurityExpert" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type frontend --name "ReactExpert" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type devops --name "DevOpsPro" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type rust --name "RustExpert" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@echo "✅ All agents started!"

stop: ## Stop all running processes (server, agents, GUI)
	@echo "🛑 Stopping all Neural Junkie processes..."
	@lsof -ti :8080 2>/dev/null | xargs kill -9 2>/dev/null || true
	@lsof -ti :1420 2>/dev/null | xargs kill -9 2>/dev/null || true
	@pkill -f "cmd/server/main.go" 2>/dev/null || true
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
	@bash -c 'source load-env.sh && go run cmd/server/main.go > /tmp/chat-server.log 2>&1 &'
	@sleep 3
	@echo "🤖 Starting all agents..."
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type backend --name "GoExpert" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 2
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type database --name "SQLMaster" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type security --name "SecurityExpert" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type frontend --name "ReactExpert" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type devops --name "DevOpsPro" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type rust --name "RustExpert" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 2
	@echo ""
	@echo "✅ System refreshed! All processes restarted with clean state."
	@echo "📊 Server logs: /tmp/chat-server.log"
	@echo ""
	@echo "🖥️  To open GUI, run: make gui"
	@echo ""

start-all: setup-env ## Start server and all agents with environment loaded
	@echo "🚀 Starting complete Neural Junkie system..."
	@bash -c 'source load-env.sh && go run cmd/server/main.go &'
	@sleep 2
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type backend --name "GoExpert" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 2
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type database --name "SQLMaster" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type security --name "SecurityExpert" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type frontend --name "ReactExpert" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type devops --name "DevOpsPro" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 1
	@bash -c 'source load-env.sh && go run cmd/agent/main.go --type rust --name "RustExpert" --model "$${OLLAMA_CODE_MODEL:-qwen2.5-coder:14b}" &'
	@sleep 2
	@echo "✅ System started! Opening GUI..."
	@cd desktop && npm run tauri:dev

demo: ## Run a complete demo
	@echo "🎬 Starting demo..."
	@echo "This will start the server and agents, then send demo messages"
	@./scripts/demo.sh

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	@rm -rf bin/
	@rm -rf *.app
	@echo "✅ Clean complete!"

test: ## Run tests
	@echo "🧪 Running tests..."
	@go test ./...

deps: ## Download dependencies
	@echo "📦 Downloading dependencies..."
	@go mod download
	@echo "✅ Dependencies downloaded!"

pull-models: ## Pull required Ollama models (code tier + utility tier)
	@echo "📥 Pulling Ollama models..."
	@echo "  Code tier: qwen2.5-coder:14b (~9GB)..."
	@ollama pull qwen2.5-coder:14b
	@echo "  Utility tier: qwen2.5:7b (~4.5GB)..."
	@ollama pull qwen2.5:7b
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


