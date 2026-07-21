# Pulse — Overview

Pulse is a social engagement marketplace: any user can post a paid campaign
asking people to repost content, and any user can browse open campaigns,
repost them, and earn a payout once their submission is approved. There's no
fixed "business" vs. "promoter" role — every account can do both.

## Architecture

```
web (Next.js) ──HTTP/JSON──> api/common (Go/Gin) ──> MongoDB (campaigns, users,
     │                              │                  wallets, transactions, …)
     │                              └──> Redis (rate limiting)
     └──> Paystack (top-ups, payouts)   └──> Paystack Transfers/Webhooks
```

- **`web`** — Next.js + Tailwind + Radix UI (shadcn/ui components). Talks to
  the API over `NEXT_PUBLIC_API_URL`.
- **`api/common`** — Go/Gin REST API. MongoDB is the primary datastore; Redis
  backs submission rate-limiting. Real-time notifications go out over SSE.
- **Payments** — Paystack: server-initiated Transactions API for wallet
  top-ups (redirect + webhook + client verify, made idempotent on the
  payment reference), and the Transfers API for promoter payouts (admin
  approval → transfer → async webhook completion).

## Core domain

- **Campaigns** — created and funded by whoever posts them; a campaign's
  wallet lock covers its budget.
- **Submissions** — a repost against a campaign, submitted with a screenshot
  and a link; reviewed by the campaign owner (approve/reject), with an
  automatic 48h hold before payout release. A user can't submit to their own
  campaign.
- **Wallet** — every user has one `wallets` document (available/pending
  balance, total earned/spent). Top-ups and withdrawals are transaction-
  logged; withdrawals require a verified payout bank account.
- **Admin** — suspend/unsuspend users, review fraud flags and social-account
  verifications, approve/reject withdrawals, delete accounts (cascades
  across every collection referencing the user — MongoDB has no foreign
  keys, so this is done explicitly).

## Where to look next

- [setup.md](setup.md) — running the stack locally.
- `CLAUDE.md` — running build/session log for AI-assisted development on
  this repo (not written for humans, but a good source of "why" history).
