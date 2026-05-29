# Collab craft ad — pipeline you can trust

**Audience:** Developers (and power users) who tried multi-agent tools and got burned by demos that break on real repos — emphasizes engineering rigor on `/collaborate`, not first-time feature discovery.

**Canonical download:** https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.19

**Regenerate graphic:**

```bash
chmod +x ./scripts/compose-collab-craft-ad.sh
./scripts/compose-collab-craft-ad.sh
./scripts/sync-gallery.sh
```

**Asset:** `assets/neural-junkie-collab-craft-ad-1080.png`

Related: [COLLABORATION.md](../COLLABORATION.md) · [COMMUNITY-AD.md](COMMUNITY-AD.md) (feedback invite) · existing orchestration ad (`compose-collaboration-ad.sh`).

---

## Headline on image

**COLLAB THAT HOLDS UP**

Sub: Planning → review → execute — tested, gated, bounded.

---

## X / LinkedIn (short)

> `/collaborate` is easy to demo. **Making it work in a real repo** is the work.
>
> Neural Junkie **beta.18**: repo binding, task parsing, multi-collab routing, workspace gates before execution, DAG dispatch — plus **`make collab-smoke`** so planning → review → execute is a pipeline, not a slide deck.
>
> If collab broke on you before, try again and tell us what still fails.
>
> https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.19

---

## LinkedIn (longer)

> Multi-agent hype usually stops at screenshots: agents talk, plan looks great, execution never lands in the right folder.
>
> We're in the **make collab hold up** phase:
> - **Bounded planning** — turn and message caps; dedicated `collab-…` channels
> - **Your approval** — review phase before anything runs
> - **Workspace gate** — tasks don't dispatch until you confirm the sandbox
> - **DAG execution** — ready tasks only; upstream output flows downstream
> - **beta.18 fixes** — source repo binding, task extraction from plans, multi-collab phase routing
> - **`make collab-smoke`** — real hub agents through plan → review → execute
>
> Open source beta (macOS / Windows / Linux): https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.19  
> Docs: https://github.com/camronwood/neural-junkie/blob/main/docs/COLLABORATION.md

---

## Demo video (ideal 15–25s)

1. `/collaborate` in a real repo channel → auto-switch to `collab-…`
2. Collaboration panel: planning → **Submit for review** → **Approve & start**
3. **Continue** on workspace gate
4. Task completes with `TASK_STATUS: completed` — show file in sandbox or worktree

Pair with a real `collab-…` screenshot in the reply thread.

---

## Reply templates

| Objection | Response |
|-----------|----------|
| "Agents still go off the rails" | Discussion is bounded; execution is gated and task-scoped. File changes still need your approval in Pending changes. |
| "How is this different from the old collab ad?" | That ad introduced orchestration. This one is **reliability** — what we fixed in beta.18 and how we test it. |
| "I only need one LLM" | Fair — use DMs. Collab is for when one thread isn't enough and you want a plan + parallel tasks under your control. |

---

## Posting order

After a release that touches collaboration. Follow with [COMMUNITY-AD.md](COMMUNITY-AD.md) if you want "try it and file issues" in the same week.
