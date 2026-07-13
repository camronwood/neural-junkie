#!/usr/bin/env bash
# Update GitHub repo descriptions, homepages, and topics for Neural Junkie + packs.
set -euo pipefail

SITE_URL="${NJ_SITE_URL:-https://www.neuraljunkie.com}"
PACKS_URL="${SITE_URL}/packs.html"
MAIN_TOPICS=(
  multi-agent
  ai-agents
  local-ai
  ollama
  tauri
  desktop-app
  open-source
  slack-integration
  lora
  neural-junkie
)

edit_repo() {
  local repo="$1"
  shift
  echo ">>> gh repo edit camronwood/${repo} $*"
  gh repo edit "camronwood/${repo}" "$@"
}

sync_main_repo() {
  edit_repo neural-junkie \
    --homepage "${SITE_URL}/" \
    --description "Multi-agent orchestration for teams and individuals — local-first AI, custom experts, Slack integration, and collaboration with human approval."
  local topic_args=()
  for topic in "${MAIN_TOPICS[@]}"; do
    topic_args+=(--add-topic "${topic}")
  done
  edit_repo neural-junkie "${topic_args[@]}"
}

sync_pack_repo() {
  local repo="$1"
  local description="$2"
  shift 2
  local topics=("$@")
  edit_repo "${repo}" --homepage "${PACKS_URL}" --description "${description}"
  local topic_args=()
  for topic in "${topics[@]}"; do
    topic_args+=(--add-topic "${topic}")
  done
  edit_repo "${repo}" "${topic_args[@]}"
}

main() {
  if ! command -v gh >/dev/null 2>&1; then
    echo "gh CLI required" >&2
    exit 1
  fi

  sync_main_repo

  sync_pack_repo neural-junkie-pack-software-development \
    "Official Neural Junkie pack — software development specialists (backend, frontend, security, architecture)" \
    neural-junkie ai-agents domain-pack software-development

  sync_pack_repo neural-junkie-pack-life-sciences \
    "Official Neural Junkie pack — life sciences workflows, lab QC, and bioinformatics" \
    neural-junkie ai-agents domain-pack life-sciences bioinformatics

  sync_pack_repo neural-junkie-pack-cad \
    "Official Neural Junkie pack — CAD and engineering design" \
    neural-junkie ai-agents domain-pack cad engineering

  sync_pack_repo neural-junkie-pack-aws \
    "Official Neural Junkie pack — AWS infrastructure and cloud operations" \
    neural-junkie ai-agents domain-pack aws devops

  sync_pack_repo neural-junkie-pack-web-browser \
    "Official Neural Junkie pack — web browsing and research agents" \
    neural-junkie ai-agents domain-pack web-browser

  sync_pack_repo neural-junkie-pack-incident-management \
    "Official Neural Junkie pack — incident response and on-call workflows" \
    neural-junkie ai-agents domain-pack incident-management sre

  sync_pack_repo neural-junkie-pack-specialist-tuning \
    "Official Neural Junkie pack — LoRA specialist training and tuning" \
    neural-junkie ai-agents domain-pack lora fine-tuning

  sync_pack_repo neural-junkie-pack-music-creation \
    "Official Neural Junkie pack — music creation and audio production" \
    neural-junkie ai-agents domain-pack music audio

  sync_pack_repo neural-junkie-pack-model-arena \
    "Official Neural Junkie pack — Model Arena (chess, Connect Four, logic puzzles)" \
    neural-junkie ai-agents domain-pack benchmarks

  sync_pack_repo neural-junkie-pack-room-chat \
    "Neural Junkie Room Chat pack — ephemeral LAN rooms on a host hub" \
    neural-junkie ai-agents domain-pack chat

  edit_repo homebrew-tap \
    --homepage "${SITE_URL}/download.html" \
    --description "Homebrew tap for Neural Junkie (macOS cask)"

  echo "Done. Set GitHub social preview image manually: Settings → General → Social preview"
  echo "  Use: docs/assets/icon/og-image.png"
}

main "$@"
