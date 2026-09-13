# Pulse — Project Summary

*Written for funding applications and for any agent/collaborator picking up this project cold. Last reviewed: 2026-09-13, against commit `2300381` on `main`.*

---

## One-liner

**Pulse is a social engagement marketplace: anyone can post a paid campaign asking people to repost/share content, and anyone can browse open campaigns, repost them, and get paid once their submission is verified.** There is no fixed "business" vs. "promoter" account type — every user can both fund campaigns and earn from them.

## The problem

Small businesses and creators want organic-feeling social reach (reposts, shares) but have no direct way to pay ordinary people for it — influencer platforms are built for accounts with large followings, not everyday users. Meanwhile there's an underused supply of people willing to repost content for modest, per-post pay, but no marketplace connects the two sides, verifies the repost actually happened, prices the payout fairly, or moves the money.

## What Pulse does

1. A user creates a **campaign** (target URL, platform, per-repost payout rate, eligibility minimums, budget) and funds it from their wallet.
2. Any other user browses the campaign marketplace ("Earn Hub"), reposts the content on Instagram or Twitter/X, and **submits proof** (post link + screenshot).
3. The campaign owner (or admin) reviews and approves/rejects the submission.
4. On approval, the payout is computed from the submitter's **influence score** (see below), the platform takes a commission (default 20%), and the rest goes to the submitter's wallet.
5. Users withdraw earnings via a **Paystack-verified bank account** — real bank transfers, not a simulated ledger.

### Anti-fraud and pricing logic (the actual IP)
Because reposts can't be verified programmatically (Instagram/Twitter don't expose repost APIs), Pulse leans on scoring and heuristics instead of trust:
- **Influence score (0–100)** per connected social account: follower count, engagement rate, account age, historical approval rate, audience quality. Payout = `baseRate × (0.5 + influenceScore/100)` — better/more credible accounts earn more per repost, which also disincentivizes fresh fraud accounts.
- **Fraud detection**: hard blocks (account age < 30 days, follower:following ratio < 0.2, duplicate repost URLs across users, >3 submissions/hour via Redis rate limiting) plus a **trust score** (0–100, starts at 50; +5/approval, -15/rejection, -30/fraud flag, auto-suspend below 20).
- **Admin panel** for platform stats, user suspension, fraud-flag review, and withdrawal approval.

### Other shipped features
- Email/password auth (JWT access+refresh, email verification, password reset) + Google OAuth.
- Real-time notifications over Server-Sent Events (submission approved/rejected, withdrawal processed, wallet top-up).
- **AI support chat** — Groq-primary/Gemini-fallback LLM assistant with embeddings for retrieval, delivered over WebSocket, that answers user questions without always escalating to a human admin.
- Marketing landing page, Terms/Privacy acceptance gate, mobile-responsive dashboard.
- Full CI/CD: GitHub Actions → Railway (API) + Vercel (web), Dockerized, health-checked.

## Tech stack

| Layer | Choice |
|---|---|
| Frontend | Next.js 14, TypeScript, Tailwind, shadcn/ui (Radix), Zustand, Recharts |
| Backend | Go + Gin, ~9.7K LOC |
| Database | MongoDB (mongo-driver v2) |
| Cache / rate limiting | Redis (sliding-window middleware) |
| Payments | Paystack — Transactions API (top-ups) + Transfers API (real bank payouts), webhook-driven, idempotent |
| Real-time | Server-Sent Events (notifications) + WebSocket (AI support chat) |
| AI | Groq (chat completions, primary) + Gemini (fallback + embeddings) |
| Email | Resend / Brevo / SMTP fallback chain |
| Infra | Docker Compose (dev), Railway (API) + Vercel (web) or self-hosted VPS via Caddy (auto-HTTPS) |

Total: ~16.5K lines of application code (9,675 Go, 6,894 TypeScript/TSX).

## Architecture

```
web (Next.js) ──HTTP/JSON──> api/common (Go/Gin) ──> MongoDB (campaigns, users, wallets, …)
     │                              │                └──> Redis (rate limiting)
     └──> Paystack (top-ups, payouts)   └──> Paystack Transfers/Webhooks
```

- **`api/common`** — the entire backend, module-per-domain (`auth`, `campaigns`, `submissions`, `wallet`, `admin`, `chat`, `notifications`), each with its own `service.go`/`handler.go`/`routes.go`/`dto.go`.
- **`web`** — Next.js App Router, role-aware dashboard, marketing site under `(marketing)`.

## Business model

Platform commission on every approved submission's payout — currently a flat 20% (`PLATFORM_COMMISSION_RATE`, configurable). Revenue scales directly with campaign volume × repost volume; no separate subscription tier exists yet.

## Current state (be precise about this for any pitch)

- **Stage:** Feature-complete MVP, not yet publicly launched — no live users, no revenue, no traction metrics yet. Built solo, AI-assisted, in a series of scoped phases (see `CLAUDE.md` for the full build log).
- **What's real vs. simulated:** Payments are real (live Paystack integration, not mocked) — top-ups and payouts move actual money once live API keys are configured. Repost verification is manual (screenshot + link review), not automated, because Instagram/Twitter/X have no public repost-verification API.
- **Deploy target:** Nigeria-first (`PAYSTACK_CURRENCY` defaults to NGN), but Paystack also supports GHS/ZAR/KES/USD, so the same codebase covers several African markets without rework.
- **Repo:** `https://github.com/siyanbolagiggs1/pulse` (private/public status not verified here), MIT-licensed, owned by Khalid Siyanbola.
- **Gaps a funder will ask about:** no automated tests mentioned beyond a CI type-check/build step; no analytics/usage dashboards for growth tracking; no formal fraud-model validation against real adversarial users yet; single-founder team as far as this repo shows.

## Where to look next

- `README.md` — setup, deployment, full API reference, environment variables.
- `docs/overview.md` — short architecture/domain summary.
- `CLAUDE.md` — the complete phase-by-phase build log and every design decision made along the way (not written for humans, but the best "why" trail in the repo).
- `api/common/internal/services/scoring/` and `.../fraud/` — the influence-scoring and fraud-detection logic described above.
- `api/common/internal/modules/chat/` and `.../services/ai/` — the AI support assistant.

## Open items before this can go into an actual funding application

These aren't in the repo and need the founder's input — don't guess at them:
- Funding ask amount and specific use of funds (infra costs, marketing, hiring, compliance/licensing for a Nigerian payments product, etc.)
- Traction/validation to date, if any (waitlist, pilot businesses, LOIs)
- Team — who else, if anyone, is involved
- Regulatory posture — operating a marketplace that moves money in Nigeria may have fintech/BSP-adjacent compliance requirements worth addressing proactively in a pitch
- Competitive landscape (how this differs from existing influencer-marketing platforms in target markets)
