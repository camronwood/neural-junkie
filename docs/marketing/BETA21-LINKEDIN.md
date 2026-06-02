# LinkedIn — v1.0.0-beta.21 (three posts)

**Do not combine audiences in one post.** Ship three separate posts (or a thread), each with its own image:

| # | Audience | Image | Copy file |
|---|----------|-------|-----------|
| 1 | **Slack users** — mobile inbox, forwarding | `assets/neural-junkie-beta21-slack-ad-1200.png` | [BETA21-SLACK-LINKEDIN.md](BETA21-SLACK-LINKEDIN.md) |
| 2 | **Builders** — chat quality, workspace honesty | `assets/neural-junkie-beta21-chat-ad-1200.png` | [BETA21-CHAT-LINKEDIN.md](BETA21-CHAT-LINKEDIN.md) |
| 3 | **Contributors** — clone repo, run scenarios | `assets/neural-junkie-beta21-test-ad-1200.png` | [BETA21-TEST-LINKEDIN.md](BETA21-TEST-LINKEDIN.md) |

**Regenerate all images:**

```bash
chmod +x ./scripts/compose-beta21-ads.sh
./scripts/compose-beta21-ads.sh all
./scripts/sync-gallery.sh
```

**Download (same for every post):** https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.21

**Posting order (suggested):** Slack → Chat → Test (widest → builder → maintainer). Space 2–4 days if you want reach without fatigue.

---

## Release post (optional fourth, text-only)

Short “beta.21 is live” with link only — no technical pillars. Point readers to the three topic posts above or to [release-notes.html](../release-notes.html).

> **v1.0.0-beta.21** — Slack inbox, sharper chat, and a bigger test harness for anyone hacking on the repo.  
> https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.21

---

## Deprecated combined creative

`assets/neural-junkie-beta21-ad-1200.png` mixed Slack + context v2 + testing in one frame. Prefer the three audience-specific assets above.
