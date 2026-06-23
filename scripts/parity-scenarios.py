#!/usr/bin/env python3
"""Run Cursor parity scenarios against a live hub."""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib import collab_hub as hub  # noqa: E402
from lib.scenario_assert import (  # noqa: E402
    check_file_deliverable,
    check_text_patterns,
    expand_deliverable_steps,
    looks_like_read_only_inspection_command,
    merge_deliverable_step,
    scenario_question,
)
from lib.workspace_context import enrich_send_metadata  # noqa: E402

SCENARIOS_DIR = ROOT / "scenarios" / "parity"
DEFAULT_CHANNEL = "parity-scenarios"
DEFAULT_FROM = "ParityScenario"


def load_scenario(name: str) -> dict:
    path = SCENARIOS_DIR / f"{name}.json"
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def reset_fixture_baseline(scenario: dict) -> None:
    ws_cfg = scenario.get("workspace") if isinstance(scenario.get("workspace"), dict) else {}
    fixture = (ws_cfg.get("fixture") or "").strip()
    if not fixture:
        return
    fixture_root = ROOT / "scenarios" / "fixtures" / fixture
    baseline = fixture_root / ".scenario-baseline"
    if not baseline.is_dir():
        return
    for src in baseline.rglob("*"):
        if not src.is_file():
            continue
        rel = src.relative_to(baseline)
        dest = fixture_root / rel
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_bytes(src.read_bytes())


def scenario_repo_root(scenario: dict) -> str:
    ws_cfg = scenario.get("workspace") if isinstance(scenario.get("workspace"), dict) else {}
    root = os.environ.get("NEURAL_JUNKIE_SCENARIO_REPO", str(ROOT)).strip()
    fixture = (ws_cfg.get("fixture") or "").strip()
    if fixture:
        root = str((ROOT / "scenarios" / "fixtures" / fixture).resolve())
    return root


class ParityContext:
    def __init__(self, base: str, scenario: dict) -> None:
        self.base = base.rstrip("/")
        self.scenario = scenario
        self.channel = (scenario.get("channel") or DEFAULT_CHANNEL).strip()
        self.target_agent = (scenario.get("target_agent") or "BackendEngineer").strip().lstrip("@")
        self.baseline_agent_count: dict[str, int] = {}
        self.file_snapshots: dict[str, str] = {}


def _chat_baseline(ctx: ParityContext, agent: str) -> int:
    msgs = hub.list_messages(ctx.base, ctx.channel, 200)
    return hub.count_chat_agent_messages(hub.chat_agent_messages(msgs), agent)


def _file_hash(root: Path, rel: str) -> str:
    path = (root / rel).resolve()
    if not path.is_file():
        return ""
    try:
        return hashlib.sha256(path.read_bytes()).hexdigest()
    except OSError:
        return ""


def snapshot_files(ctx: ParityContext, rel_paths: list[str]) -> None:
    root = Path(scenario_repo_root(ctx.scenario))
    for rel in rel_paths:
        rel = str(rel).strip()
        if rel:
            ctx.file_snapshots[rel] = _file_hash(root, rel)


def step_send(ctx: ParityContext, step: dict) -> tuple[bool, str]:
    from_name = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    ctx.baseline_agent_count[from_name] = _chat_baseline(ctx, from_name)
    content = (step.get("content") or "").strip()
    meta = enrich_send_metadata(step.get("metadata"), ctx.scenario, content=content)
    code, _ = hub.send_message(ctx.base, ctx.channel, content, metadata=meta, from_name=DEFAULT_FROM)
    return (True, "sent") if code == 200 else (False, f"send failed ({code})")


def step_wait_reply(ctx: ParityContext, step: dict) -> tuple[bool, str]:
    timeout = step.get("timeout", "420s")
    secs = 420
    if isinstance(timeout, str) and timeout.endswith("s"):
        try:
            secs = int(timeout[:-1])
        except ValueError:
            pass
    from_name = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    until_any = step.get("until_any_match")
    baseline = int(step.get("baseline", ctx.baseline_agent_count.get(from_name, _chat_baseline(ctx, from_name))))
    deadline = time.time() + secs
    while time.time() < deadline:
        msgs = hub.list_messages(ctx.base, ctx.channel, 200)
        pool = hub.chat_agent_messages(msgs)
        for msg in [m for m in pool if m.get("from", {}).get("name") == from_name][baseline:]:
            text = msg.get("content") or ""
            if until_any:
                ok, detail = check_text_patterns(text, any_match=until_any)
                if ok:
                    return True, f"reply from {from_name} ({detail})"
            else:
                return True, f"reply from {from_name}"
        time.sleep(2)
    return False, f"timeout waiting for {from_name}"


def step_assert_messages(ctx: ParityContext, step: dict) -> tuple[bool, str]:
    msgs = hub.list_messages(ctx.base, ctx.channel, 40)
    from_name = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    candidates = [m for m in msgs if m.get("from", {}).get("name") == from_name]
    if not candidates:
        return False, f"no messages from {from_name}"
    text = candidates[-1].get("content") or ""
    if any_match := step.get("any_match"):
        ok, detail = check_text_patterns(text, any_match=any_match)
        if not ok:
            return False, detail
    if none_match := step.get("none_match"):
        ok, detail = check_text_patterns(text, none_match=none_match)
        if not ok:
            return False, detail
    return True, "message assertions ok"


def step_assert_deliverable(ctx: ParityContext, step: dict) -> tuple[bool, str]:
    root = scenario_repo_root(ctx.scenario)
    spec = merge_deliverable_step(ctx.scenario, step)
    return check_file_deliverable(
        root=root,
        rel=(spec.get("path") or "").strip(),
        spec=spec,
        question=scenario_question(ctx.scenario),
        hub_base=ctx.base,
    )


def step_assert_file_exists(ctx: ParityContext, step: dict) -> tuple[bool, str]:
    root = Path(scenario_repo_root(ctx.scenario))
    rel = (step.get("path") or "").strip()
    if not rel:
        return False, "path required"
    path = root / rel
    if not path.is_file():
        return False, f"missing {rel}"
    if contains := step.get("contains"):
        body = path.read_text(encoding="utf-8", errors="replace")
        if contains not in body:
            return False, f"{rel} missing {contains!r}"
    return True, rel


def step_api_semantic_search(ctx: ParityContext, step: dict) -> tuple[bool, str]:
    root = scenario_repo_root(ctx.scenario)
    query = (step.get("query") or "").strip()
    body = json.dumps({"repo_path": root, "query": query, "limit": 12}).encode()
    url = f"{ctx.base}/api/repo/search/semantic"
    req = urllib.request.Request(url, data=body, method="POST", headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=180) as resp:
            data = json.loads(resp.read().decode())
    except (urllib.error.HTTPError, OSError) as e:
        return False, str(e)
    results = data.get("results") if isinstance(data, dict) else data
    if not isinstance(results, list):
        results = []
    expect_path = (step.get("expect_path_contains") or "").lower()
    expect_content = step.get("expect_content_contains") or ""
    for item in results:
        path = str(item.get("path") or "").lower()
        content = str(item.get("content") or "")
        if expect_path and expect_path not in path:
            continue
        if expect_content and expect_content not in content:
            continue
        return True, f"semantic hit {path}"
    # grep fallback
    grep_url = f"{ctx.base}/api/workspace/search?q={urllib.parse.quote(query)}&workspace_path={urllib.parse.quote(root)}"
    try:
        with urllib.request.urlopen(grep_url, timeout=60) as resp:
            grep_data = json.loads(resp.read().decode())
        for hit in grep_data.get("results") or grep_data.get("paths") or []:
            p = str(hit.get("path") if isinstance(hit, dict) else hit)
            if expect_path in p.lower() or expect_content in p:
                return True, f"grep fallback {p}"
    except OSError:
        pass
    return False, f"no match in {len(results)} semantic results"


def step_assert_shell(ctx: ParityContext, step: dict) -> tuple[bool, str]:
    root = scenario_repo_root(ctx.scenario)
    cmd = (step.get("command") or "").strip()
    expect_exit = int(step.get("expect_exit", 0))
    proc = subprocess.run(cmd, shell=True, cwd=root, capture_output=True, text=True)
    if proc.returncode != expect_exit:
        out = ((proc.stdout or "") + (proc.stderr or ""))[-800:]
        return False, f"exit {proc.returncode} want {expect_exit}: {out}"
    return True, f"exit {proc.returncode}"


HANDLERS = {
    "send": step_send,
    "wait_reply": step_wait_reply,
    "assert_messages": step_assert_messages,
    "assert_deliverable": step_assert_deliverable,
    "assert_file_exists": step_assert_file_exists,
    "api_semantic_search": step_api_semantic_search,
    "assert_shell": step_assert_shell,
}


def ensure_channel(ctx: ParityContext) -> bool:
    required = ctx.scenario.get("required_agents") or [ctx.target_agent]
    ok, _ = hub.ensure_channel_with_agents(
        ctx.base, ctx.channel, required, ctx.scenario.get("description") or "parity scenarios"
    )
    return ok


def run_scenario(base: str, name: str, *, keep: bool = False) -> bool:
    scenario = load_scenario(name)
    ctx = ParityContext(base, scenario)
    print(f"\n=== parity: {name} ===")
    if not hub.check_health(base):
        print("  FAIL: hub unhealthy", file=sys.stderr)
        return False
    required = scenario.get("required_agents") or [ctx.target_agent]
    ok, missing = hub.verify_agents_online(base, required)
    if not ok:
        print(f"  FAIL: offline agents: {missing}", file=sys.stderr)
        return False
    if not ensure_channel(ctx):
        return False
    if scenario.get("cleanup", "clear") == "clear" and not keep:
        hub.clear_channel_history(ctx.base, ctx.channel)
    reset_fixture_baseline(scenario)
    time.sleep(3.0)
    all_ok = True
    steps = list(scenario.get("setup") or []) + list(scenario.get("steps") or []) + expand_deliverable_steps(scenario)
    for i, step in enumerate(steps, 1):
        action = (step.get("action") or "").strip()
        fn = HANDLERS.get(action)
        if not fn:
            print(f"  FAIL unknown action {action}", file=sys.stderr)
            all_ok = False
            break
        ok, detail = fn(ctx, step)
        print(f"  {'✓' if ok else '✗'} [{i}] {action}: {detail}")
        if not ok:
            all_ok = False
            break
    if scenario.get("cleanup", "clear") == "clear" and not keep:
        hub.clear_channel_history(ctx.base, ctx.channel)
    print(f"=== {'PASS' if all_ok else 'FAIL'}: {name} ===\n")
    return all_ok


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--list", action="store_true")
    p.add_argument("--scenario")
    p.add_argument("--all", action="store_true")
    p.add_argument("--hub", default=hub.DEFAULT_HUB)
    p.add_argument("--keep", action="store_true")
    args = p.parse_args()
    if args.list:
        for f in sorted(SCENARIOS_DIR.glob("*.json")):
            print(f.stem)
        return 0
    names = [f.stem for f in sorted(SCENARIOS_DIR.glob("*.json"))] if args.all else ([args.scenario] if args.scenario else [])
    if not names:
        p.print_help()
        return 1
    failed = sum(1 for n in names if not run_scenario(args.hub, n, keep=args.keep))
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
