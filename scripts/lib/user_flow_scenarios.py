"""Real-world user-flow suite — product journeys a person would paste into NJ.

Kept under scenarios/user-flows/ so they do not inflate the implement 20/20 or
collab-core gates. Reused core scenarios are referenced by kind+name only.
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class UserFlowEntry:
    """One suite member: run via implement or collab runner."""

    name: str
    kind: str  # "implement" | "collab"
    source: str  # "user-flows" | "core"
    description: str = ""
    # If set, --all skips this entry (still listed; --scenario can force-run).
    skip_reason: str = ""


_WIP = "still landing; not release-blocking (force with --scenario)"


# Canonical order: research → games/APIs → websites → fix → multi-turn journeys.
USER_FLOW_SCENARIOS: tuple[UserFlowEntry, ...] = (
    UserFlowEntry(
        name="trip-research-vacation",
        kind="implement",
        source="user-flows",
        description="Research STL→Seaside FL trip into vacation_2026.md",
        skip_reason="mapping / trip-research tooling not landed yet",
    ),
    UserFlowEntry(
        name="rust-blackjack-2d",
        kind="implement",
        source="user-flows",
        description="Local 2D/CLI Rust blackjack against the house",
        skip_reason=_WIP,
    ),
    UserFlowEntry(
        name="nodejs-user-crud",
        kind="implement",
        source="user-flows",
        description="Node.js + TypeScript user CRUD API with seed users",
    ),
    UserFlowEntry(
        name="ios-trivia-swift",
        kind="implement",
        source="user-flows",
        description="Local Swift trivia game under Xcode-like structure",
        skip_reason=_WIP,
    ),
    UserFlowEntry(
        name="collaboration-station-branded",
        kind="collab",
        source="user-flows",
        description="Static Collaboration Station site (brand colors, 3 pages)",
    ),
    UserFlowEntry(
        name="admin-cms-website",
        kind="collab",
        source="user-flows",
        description="Sample site + API + admin login/content control",
    ),
    UserFlowEntry(
        name="vite-boot-fix-corrupt-appjs",
        kind="implement",
        source="core",
        description="Real user: open workspace app won't boot — find and fix",
    ),
    # Multi-turn journeys: back-and-forth until a goal completes (clarify / correct / finish).
    UserFlowEntry(
        name="journey-crud-clarify-correct",
        kind="implement",
        source="user-flows",
        description="Vague user API → Node/TS constraints → /health correction → finish",
        skip_reason=_WIP,
    ),
    UserFlowEntry(
        name="journey-blackjack-cli-correction",
        kind="implement",
        source="user-flows",
        description="Plan Rust blackjack → implement → CLI-only correction → finish",
        skip_reason=_WIP,
    ),
    UserFlowEntry(
        name="journey-boot-fix-then-feature",
        kind="implement",
        source="user-flows",
        description="Fix corrupt Vite App.js → add Workspace Ready heading",
        skip_reason=_WIP,
    ),
    UserFlowEntry(
        name="journey-notes-rename-to-memos",
        kind="implement",
        source="user-flows",
        description="Notes CRUD → rename to /memos mid-session → scrub /notes",
        skip_reason=_WIP,
    ),
    UserFlowEntry(
        name="journey-landing-brand-correction",
        kind="implement",
        source="user-flows",
        description="Landing page → brand rename → tagline → finish",
        skip_reason=_WIP,
    ),
)


def user_flow_scenarios(*, include_skipped: bool = True) -> list[UserFlowEntry]:
    if include_skipped:
        return list(USER_FLOW_SCENARIOS)
    return [e for e in USER_FLOW_SCENARIOS if not e.skip_reason]


def user_flow_names(
    *,
    kind: str | None = None,
    source: str | None = None,
    include_skipped: bool = True,
) -> list[str]:
    out: list[str] = []
    for entry in user_flow_scenarios(include_skipped=include_skipped):
        if kind and entry.kind != kind:
            continue
        if source and entry.source != source:
            continue
        out.append(entry.name)
    return out
