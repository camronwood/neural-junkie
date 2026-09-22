You are fixing failures from Neural Junkie **layer gate: user-flows**.
Layer goal: Away primary — real-world product journeys (overnight bug hunt)
Do not weaken assertions. Fix product/hub/agent behavior first (docs/TESTING.md).
After edits, run the targeted verification commands in this brief.

---


Rules (mandatory):
- Triage product/hub/agent behavior first, harness second (docs/TESTING.md).
- Do NOT weaken test assertions or scenario contracts to greenwash flakes.
- Prefer minimal, focused fixes in the neural-junkie repo.
- After edits, run the targeted verification commands listed below.
- Summarize what you changed and which commands you ran.

Release prep summary: /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-user-flows-2026-09-22-0505-iter1.md
Failed phases: user-flow-scenarios

## Failures to address

### collab:collaboration-station-branded [code]
```text
ning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=4)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] assert_collab: collab snapshot ok
  ✓ [9] send: /resume-plan 5605baf7-48d4-424e-889b-ddfedc339284
  ✓ [10] wait_tasks: tasks completed
  ✓ [11] send: /resume-plan 5605baf7-48d4-424e-889b-ddfedc339284
  ✗ [12] wait_tasks: task wait timeout statuses=['completed', 'completed', 'in_progress', 'in_progress']

  --- transcript (agent messages) ---
    [collaboration_discussion] Claude: ``` [FILE_CHANGE] operation: create path: collabs/5605baf7-48d4-424e-889b-ddfedc339284/about.html ```new <!DOCTYPE html> <html lang="en"> <head>     <meta chars
    [collaboration_discussion] FrontendEngineer: I submitted the file changes as proposals; they have not been applied yet.
    [collaboration_discussion] FrontendEngineer: I submitted the file changes as proposals; they have not been applied yet.
    [collaboration_discussion] Claude: ``` [FILE_CHANGE] operation: create path: collabs/5605baf7-48d4-424e-889b-ddfedc339284/contact.html ```new <!DOCTYPE html> <html lang="en"> <head>     <meta cha
    [collaboration_discussion] Claude: **Write collabs/5605baf7-48d4-424e-889b-ddfedc339284/about.html using style.css**  ```html <!DOCTYPE html> <html lang="en"> <head>     <meta charset="UTF-8">   
    [collaboration_discussion] Claude: ``` [FILE_CHANGE] operation: create path: collabs/5605baf7-48d4-424e-889b-ddfedc339284/about.html ```new <!DOCTYPE html> <html lang="en"> <head>     <meta chars
    [collaboration_discussion] Claude: ``` [FILE_CHANGE] operation: create path: collabs/5605baf7-48d4-424e-889b-ddfedc339284/contact.html ```new <!DOCTYPE html> <html lang="en"> <head>     <meta cha
    [collaboration_discussion] Claude: ``` [FILE_CHANGE] operation: create path: collabs/5605baf7-48d4-424e-889b-ddfedc339284/contact.html ```new <!DOCTYPE html> <html lang="en"> <head>     <meta cha
  --- end ---

=== FAIL: collaboration-station-branded ===
```

## Likely flakes/infra (retry only — do not hack code to pass)

collab:rust-blackjack-2d, collab:nodejs-user-crud, collab:ios-trivia-swift, collab:journey-crud-clarify-correct, collab:journey-blackjack-cli-correction, collab:journey-boot-fix-then-feature, collab:journey-notes-rename-to-memos

## Child artifacts (read for full context)
- /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-user-flows-2026-09-22-0505-iter1.log

## Targeted verification (run after your fixes)
- python3 scripts/collab-scenarios.py --scenario collaboration-station-branded
- python3 scripts/collab-scenarios.py --scenario ios-trivia-swift
- python3 scripts/collab-scenarios.py --scenario journey-blackjack-cli-correction
- python3 scripts/collab-scenarios.py --scenario journey-boot-fix-then-feature
- python3 scripts/collab-scenarios.py --scenario journey-crud-clarify-correct
- python3 scripts/collab-scenarios.py --scenario journey-notes-rename-to-memos
- python3 scripts/collab-scenarios.py --scenario nodejs-user-crud
- python3 scripts/collab-scenarios.py --scenario rust-blackjack-2d

