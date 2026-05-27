# Pulse — Project Briefing for Claude

> Read this file at the start of every session before writing any code.
> Update this file at the end of every session before stopping.

---

## What Is Pulse

A social engagement marketplace MVP.
- Businesses create repost campaigns with budgets and payout rules
- Promoters (everyday users) earn money by reposting on Instagram and Twitter/X
- Platform takes a configurable commission (default 20%)
- Admin manages everything

---

## Tech Stack

| Layer | Choice |
|---|---|
| Frontend | Next.js 14, TypeScript, TailwindCSS, shadcn/ui, Recharts |
| Backend | Go + Gin |
| Database | MongoDB (mongo-driver/v2) |
| Cache / Rate limiting | Redis |
| Payments | Stripe (Payment Intents for top-ups, Stripe Connect for payouts) |
| Real-time | SSE — Server-Sent Events (replaces Socket.IO; Go-native, sufficient for push notifications) |
| Email | Go SMTP (net/smtp or gomail) |
| Infrastructure | Docker + Docker Compose |

---

## Monorepo Structure

```
/pulse
├── apps/
│   ├── web/          # Next.js 14 frontend
│   └── api/          # Node.js/Express backend
├── docker-compose.yml
├── .env.example
└── CLAUDE.md
```

---

## Roles

- `admin` — manages platform, approves submissions/withdrawals, handles fraud
- `business` — creates campaigns, funds wallet, reviews submissions
- `promoter` — connects social accounts, joins campaigns, earns rewards

---

## Core Flow (Proof of Repost)

1. Business creates campaign with budget + base payout rate
2. Promoter accepts campaign (must meet follower/engagement minimums)
3. Promoter reposts manually on Instagram or Twitter/X
4. Promoter submits: post URL + screenshot
5. Admin reviews and approves or rejects
6. On approval: business wallet locked → platform commission deducted → promoter pending balance
7. After 48h hold: promoter pending → available balance
8. Promoter withdraws via Stripe Connect

---

## Payout Formula

```
influenceMultiplier = 0.5 + (influenceScore / 100)
finalPayout = baseRepostRate × influenceMultiplier
```

Influence score range: 0–100, based on:
- followerScore (max 30)
- engagementScore (max 25)
- accountAgeScore (max 15)
- completionScore (max 20)
- audienceQualityScore (max 10)

---

## Fraud Rules

Hard blocks:
- Account age < 30 days → ineligible
- follower:following ratio < 0.2 → flagged
- Same repost URL submitted by multiple users → blocked
- > 3 campaign submissions in 1 hour → rate limited

Trust score: 0–100, starts at 50
- +5 per approval, -15 per rejection, -30 per fraud flag
- Trust score < 20 → auto-suspend pending review

---

## API Base URL

- Dev: `http://localhost:5000/api`
- All responses: `{ success: boolean, data?: any, message?: string, errors?: [] }`

---

## Environment Variables

See `apps/api/.env.example` and `apps/web/.env.example` once created.

---

## Build Phases

| Phase | Scope | Status |
|---|---|---|
| 1 | Project scaffold, Docker, MongoDB schemas, base Go/Gin server, folder structure | ✅ Complete |
| 2 | Auth system: register, login, JWT (access + refresh), email verification, password reset | ✅ Complete |
| 3 | Campaign CRUD (business), campaign model, business dashboard APIs | ✅ Complete |
| 4 | Campaign marketplace (promoter), submission flow, proof upload | ✅ Complete |
| 5 | Wallet system + Stripe integration (top-up + Connect payouts) | ✅ Complete |
| 6 | Admin panel: submission review, withdrawal approval, user management | ✅ Complete |
| 7 | Influence scoring service + fraud detection service | ✅ Complete |
| 8 | Real-time notifications (SSE) | ✅ Complete |
| 9 | Frontend: Next.js pages and dashboards | ✅ Complete |
| 10 | Production deploy config (CI/CD, Fly.io, Vercel, VPS) | ✅ Complete |
| 11 | Polish + missing UX (profile, social accounts, campaign edit, pagination, mobile nav) | ✅ Complete |

---

## Current Status

**Last session:** Session 11 — Phase 11 complete. Profile page, social account management, campaign edit, pagination controls, mobile sidebar.

**Next action:** Optional Phase 12: E2E tests (Playwright), campaign analytics charts, or additional hardening.

---

## Phase 11 — Files Created

```
apps/web/src/
  components/ui/sheet.tsx               Sheet/drawer built on @radix-ui/react-dialog — slides from left; used for mobile sidebar
  components/ui/pagination.tsx          Reusable Pagination component (prev/next + numbered pages with ellipsis)

  components/layout/sidebar.tsx         Refactored: shared NavContent component; Sidebar (desktop, hidden on mobile); MobileSidebar (Sheet-based, visible on mobile); Profile link added to all nav arrays
  components/layout/header.tsx          Added hamburger Menu button (md:hidden) wired to onMenuClick prop; brand name shown on mobile
  app/dashboard/layout.tsx              Added mobileOpen state; passes onMenuClick to Header; renders MobileSidebar

  app/dashboard/profile/page.tsx        Profile page: update display name; for promoters: list/delete connected social accounts, connect new account dialog (all fields: platform, username, profileUrl, followers, following, engagement, age)

  app/dashboard/campaigns/page.tsx      Replaced confirm() delete with Dialog; added Edit icon button; wired Pagination (PAGE_SIZE=12)
  app/dashboard/campaigns/[id]/page.tsx Added Edit button (hidden for completed/cancelled); imports Pencil icon
  app/dashboard/campaigns/[id]/edit/    NEW — pre-filled campaign edit form (title, description, targetUrl, payout, eligibility, endDate); platform+budget locked

  app/dashboard/submissions/page.tsx    Wired Pagination (PAGE_SIZE=20); status filter resets page to 1
  app/dashboard/admin/users/page.tsx    Wired Pagination (PAGE_SIZE=20); role filter resets page to 1
```

Key behaviours:
- MobileSidebar closes on any nav link click (`onNavigate` callback) — prevents stale open drawer after navigation
- Pagination resets to page 1 on filter change (status, role) to avoid empty results on invalid page
- `getPageNumbers()` returns ellipsis strings for large page counts — renders ≤7 buttons at all times
- Campaign edit page uses `format(date, "yyyy-MM-dd")` to pre-fill `<input type="date">` correctly
- Social account connect form validates required fields client-side before API call; engagementRate and accountAgeDays are optional (default 0)
- Sidebar `hidden md:flex` / hamburger `md:hidden` — clean responsive breakpoint with no layout shift on desktop

## Phase 10 — Files Created

```
.gitignore                                    Root gitignore (env files, node_modules, .next, uploads)
Caddyfile                                     Caddy reverse proxy — /api/* → Go, /* → Next.js; automatic HTTPS via Let's Encrypt; security headers
docker-compose.prod.yml                       Production compose: no exposed DB ports, Caddy for HTTPS, Redis password, web build args

apps/api/
  Dockerfile                                  Updated: CGO_ENABLED=0 GOOS=linux -ldflags="-s -w", non-root user, HEALTHCHECK
  .dockerignore                               Excludes .env, uploads/, .git
  fly.toml                                    Fly.io app config: iad region, 256MB shared VM, persistent volume for uploads, health check on /health
  .env.example                                Updated: NODE_ENV production comment, REDIS_PASSWORD, CLIENT_URL prod notes

apps/web/
  Dockerfile                                  Updated: NEXT_TELEMETRY_DISABLED=1, ARG for build-time NEXT_PUBLIC_* env vars
  .dockerignore                               Excludes node_modules/, .next/, .env*
  vercel.json                                 Vercel config: framework=nextjs, security headers (X-Frame-Options, nosniff, Referrer-Policy)

.github/workflows/
  ci.yml                                      CI on all pushes/PRs: Go vet+build, Node type-check+build (with placeholder env vars)
  deploy.yml                                  CD on push to main: Fly.io API deploy then Vercel web deploy (sequential, concurrency-guarded)
```

Key behaviours:
- API binary shrinks ~35% from `-ldflags="-s -w"` (strips DWARF debug info + symbol table); static binary via `CGO_ENABLED=0` runs in scratch-compatible environments
- Non-root user (`pulse`) in API container — image fails to start if `/app/uploads` isn't chowned (handled in Dockerfile)
- Caddy automatically provisions and renews Let's Encrypt certs on first start — zero cert management
- `docker-compose.prod.yml` uses `--env-file apps/api/.env.prod` so secrets never touch `docker-compose.yml` itself
- Web build args (`NEXT_PUBLIC_API_URL`, `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY`) baked into Next.js static bundle at image build time — set correct values before `docker-compose ... up --build`
- GitHub Actions CI caches Go modules (by `go.sum`) and npm packages (by `package-lock.json`) to keep runs under 2 min
- Deploy workflow uses `concurrency: group: deploy, cancel-in-progress: false` — queues rather than cancels simultaneous deploys to `main`
- Fly.io persistent volume (`pulse_uploads`) survives deploys; mount at `/app/uploads` matches `UPLOAD_DIR` env var

## Phase 9 — Files Created

```
apps/web/src/
  store/auth.ts                                 Zustand auth store (sessionStorage token persistence)
  lib/api.ts                                    All API calls grouped by domain (authApi, usersApi, campaignsApi, submissionsApi, walletApi, adminApi, notificationsApi)
  hooks/use-sse.ts                              SSE hook using fetch + ReadableStream (NOT EventSource — cannot set Authorization header)
  types/index.ts                                TypeScript types for all domain objects

  components/ui/                                shadcn/ui Radix components (button, input, card, badge, label, textarea, separator, skeleton, avatar, tabs, select, dialog, table, dropdown-menu, toast, use-toast, toaster)
  components/layout/sidebar.tsx                 Role-based nav (businessNav / promoterNav / adminNav), active state via usePathname
  components/layout/header.tsx                  Notification bell + unread badge, SSE-powered dropdown with recent notifications

  app/layout.tsx                                Root layout with Toaster
  app/page.tsx                                  Client-side auth check → redirect to /dashboard or /login
  app/(auth)/layout.tsx                         Centered card layout with Pulse branding
  app/(auth)/login/page.tsx                     react-hook-form + zod, setAuth + role-based redirect
  app/(auth)/register/page.tsx                  Role select (business/promoter), register flow
  app/(auth)/forgot-password/page.tsx           Email form → success state
  app/(auth)/reset-password/[token]/page.tsx    Password + confirm with zod refine

  app/verify-email/[token]/page.tsx             Auto-verifies on load; success redirects to /login after 3s

  app/dashboard/layout.tsx                      Auth guard: checks token → authApi.me() → redirect to /login if unauthenticated
  app/dashboard/page.tsx                        Role-based redirect (admin/campaigns/marketplace)

  app/dashboard/campaigns/page.tsx              Business campaign grid, status badges, delete with confirm dialog
  app/dashboard/campaigns/new/page.tsx          Campaign creation form (all fields, dates, eligibility)
  app/dashboard/campaigns/[id]/page.tsx         Campaign stats + submissions table with approve/reject actions

  app/dashboard/marketplace/page.tsx            Promoter campaign browse with search + platform filter
  app/dashboard/marketplace/[id]/page.tsx       Campaign detail, eligible accounts filter, screenshot upload, submit

  app/dashboard/submissions/page.tsx            Shared submissions table (status filter, role-scoped columns)

  app/dashboard/wallet/page.tsx                 Balance cards, topup/withdraw dialogs, Connect Stripe, transaction + withdrawal history
  app/dashboard/wallet/connect/complete/page.tsx  Stripe Connect return handler → syncs status → redirects
  app/dashboard/wallet/connect/refresh/page.tsx   Stripe Connect refresh → re-calls createConnect → new URL

  app/dashboard/admin/page.tsx                  Platform stats: 8 KPI cards + submission/campaign breakdown bars
  app/dashboard/admin/users/page.tsx            User list with role filter, suspend/unsuspend actions
  app/dashboard/admin/submissions/page.tsx      Submission review queue: approve (icon) / reject (reason dialog)
  app/dashboard/admin/fraud-flags/page.tsx      Fraud flag list with resolve action
  app/dashboard/admin/withdrawals/page.tsx      Withdrawal approval queue: approve / reject actions
```

Key behaviours:
- SSE uses `fetch()` with `Authorization: Bearer <token>` + manual ReadableStream line parsing (EventSource cannot send custom headers)
- Auth guard in dashboard layout re-fetches `/auth/me` on every navigation to stay in sync with server-side session state
- Admin submissions page uses the shared `/submissions` endpoint (role-scoped server-side) so no separate admin-only submissions route needed
- Notifications header dropdown reads `unreadCount` from the paginated `GET /notifications` meta field — no extra round trip
- All forms use react-hook-form + zod for validation; shadcn/ui components are Radix UI primitives written manually (no CLI)

## Phase 8 — Files Created

```
apps/api/
  internal/services/sse/hub.go              SSE Hub singleton (sync.RWMutex map of userID→buffered chan); Register (replaces on reconnect), Unregister (closes channel), Push (non-blocking drop on full)
  internal/modules/notifications/dto.go     NotificationResponse, NotifListMeta (includes unreadCount), toNotificationResponse mapper
  internal/modules/notifications/service.go Send (persist to DB + Push via SSE), getNotifications (paginated + unread count), markAsRead, markAllAsRead
  internal/modules/notifications/handler.go handleListNotifications, handleMarkAsRead, handleMarkAllAsRead, handleStream (SSE: connected ping + 30s heartbeat + event loop)
  internal/modules/notifications/routes.go  GET /stream, POST /read-all (static before /:id), GET "", POST /:id/read

  internal/router/router.go                 Updated — notifications module wired

  internal/modules/submissions/service.go   Added: notifications.Send after approveSubmission (NotifSubmissionApproved) and rejectSubmission (NotifSubmissionRejected)
  internal/modules/admin/service.go         Added: notifications.Send after approveWithdrawal (NotifWithdrawalProcessed approved) and rejectWithdrawal (NotifWithdrawalProcessed rejected)
  internal/modules/wallet/service.go        Added: notifications.Send after creditWallet (NotifWalletTopup)
```

Key behaviours:
- SSE hub: one channel per connected user; channel is buffered (32 events); slow/disconnected clients get events dropped (never blocks the server)
- On reconnect: existing channel is closed and replaced so no stale goroutines leak
- `notifications.Send` is always called as `go notifications.Send(...)` — zero latency added to any request
- SSE stream sends `event: connected` immediately on connect, `: heartbeat` every 30s (keeps proxies from dropping idle connections), `event: notification` for each pushed notification
- `GET /notifications` returns the list AND an `unreadCount` in the meta — frontend can show a badge without a separate request
- Notification types triggered: submission approved/rejected, withdrawal approved/rejected, wallet top-up

## Phase 7 — Files Created

```
apps/api/
  internal/services/scoring/scoring.go   ScoreFollowers/Engagement/Age/AudienceQuality/Round2, ComputeCompletionScore (DB query: approved÷total × 20, neutral 10 if no history), ComputeFullScore, RefreshAllAccounts (recomputes + persists all social accounts for a promoter)
  internal/services/fraud/fraud.go       FlagUser (insert FraudFlag + trust -30 + auto-suspend < 20), CheckSubmission (follower ratio < 0.2 → FraudLowFollowerRatio; engagement > 50% with > 10k followers → FraudAbnormalEngagement), CheckAccount (reuses CheckSubmission checks)

  internal/modules/users/service.go      Refactored: removed local scoring functions (scoreFollowers etc.), replaced with scoring.* calls; connectSocialAccount uses scoring.ComputeFullScore; getInfluenceScore uses dynamic scoring.ComputeCompletionScore; triggers go fraud.CheckAccount after connect
  internal/modules/submissions/service.go Refactored: replaced inline recordFraudFlag + fraud check with fraud.CheckSubmission; removed local round2 (uses scoring.Round2); approveSubmission + rejectSubmission both trigger go scoring.RefreshAllAccounts
  internal/database/indexes.go           Added: fraud_flags (userId, resolved, userId+resolved compound), social_accounts (userId, userId+platform unique)
```

Key behaviours:
- Completion score (0–20, max component) is dynamic: computed from approved÷(approved+rejected) submissions; 10 if no history
- Influence score is recomputed and persisted to social_accounts after every approval or rejection
- `fraud.CheckSubmission` runs synchronously at submission time but all FlagUser calls are goroutines — zero latency impact
- `fraud.CheckAccount` is called as a goroutine after social account connect — onboarding never delayed
- Two fraud heuristics now active: low follower ratio (< 0.2) and abnormal engagement (> 50% with > 10k followers)
- DB indexes added for fraud_flags and social_accounts (including unique compound index on userId+platform to enforce single-account-per-platform at the DB level)

## Phase 6 — Files Created

```
apps/api/
  internal/modules/admin/dto.go       PlatformStats (nested: UserStats, CampaignStats, SubmissionStats, FinancialStats), AdminUserResponse, FraudFlagResponse, WithdrawalAdminResponse, query/request types, mappers
  internal/modules/admin/service.go   getPlatformStats (parallel counts + aggregation pipelines), listUsers, getUser, suspendUser, unsuspendUser, listFraudFlags, resolveFraudFlag, listWithdrawals, approveWithdrawal (Stripe Transfer), rejectWithdrawal (refunds wallet)
  internal/modules/admin/handler.go   HTTP handlers for all admin routes
  internal/modules/admin/routes.go    GET+POST /admin/stats|users|fraud-flags|withdrawals (all admin-role guarded)

  internal/router/router.go           Updated — admin module wired
  internal/modules/wallet/service.go  Modified — removed Stripe Transfer from requestWithdrawal (withdrawal stays pending until admin approves)
```

Key behaviours:
- All admin routes require `RequireAuth() + RequireRole("admin")`
- Stats: parallel MongoDB CountDocuments + aggregation pipelines for financial totals (no N+1)
- Withdrawal flow: promoter requests → pending (balance deducted immediately); admin approves → Stripe Transfer fires; admin rejects → balance refunded + TxRefund transaction recorded
- approveWithdrawal: gracefully skips Stripe Transfer when STRIPE_SECRET_KEY not set (dev mode)
- suspendUser: creates FraudFlag audit record with reason "admin_suspension"
- unsuspendUser: resets trustScore to 50 (neutral)

## Phase 5 — Files Created

```
apps/api/
  internal/modules/wallet/dto.go       WalletResponse, TxResponse, TopupRequest/Response, WithdrawRequest, WithdrawalResponse, ConnectOnboardingResponse, ConnectStatusResponse + mappers
  internal/modules/wallet/service.go   getWallet (lazy 48h release on read), getTransactions, createTopup (Stripe Payment Intent), creditWallet (webhook handler), requestWithdrawal (Stripe Transfer), getWithdrawals, createConnectAccount (Express onboarding), getConnectStatus, releaseMaturePending
  internal/modules/wallet/handler.go   HTTP handlers
  internal/modules/wallet/routes.go    GET /wallet, GET /wallet/transactions, POST /wallet/topup, POST /wallet/topup/webhook (no auth), POST /wallet/withdraw, GET /wallet/withdrawals, POST /wallet/connect, GET /wallet/connect/status

  internal/router/router.go            Updated — wallet module wired
  internal/models/submission.go        Added PayoutReleased bool field for 48h release tracking
```

Dependencies added: github.com/stripe/stripe-go/v78 v78.12.0

Key behaviours:
- Top-up: creates Stripe Payment Intent; wallet credited only after Stripe fires payment_intent.succeeded webhook (signature verified)
- Withdrawal: checks available balance ≥ amount ≥ $10; deducts balance; creates Withdrawal record; initiates Stripe Transfer to Connect account; rolls back on Stripe failure
- Connect onboarding: creates Express account if none exists; returns onboarding URL; syncs status back to user on status check
- 48h release: on every GET /wallet, releaseMaturePending scans for approved submissions past their payoutReleasedAt with payoutReleased=false; moves earnings to availableBalance + totalEarned; marks submissions released; records TxPayoutReleased transaction
- Stripe not configured: all Stripe-dependent endpoints return 503 gracefully (dev mode without keys still works for balance reads)

## Phase 4 — Files Created

```
apps/api/
  internal/modules/submissions/dto.go      CreateSubmissionRequest, RejectRequest, SubmissionResponse, SubmissionListQuery, UploadResponse + mapper
  internal/modules/submissions/service.go  createSubmission (eligibility + fraud + rate limit), getSubmissions (role-scoped), getSubmission, approveSubmission (wallet ops + trust score), rejectSubmission (trust score + slot release), saveScreenshot
  internal/modules/submissions/handler.go  HTTP handlers
  internal/modules/submissions/routes.go   POST /submissions, POST /submissions/upload, GET /submissions, GET /submissions/:id, POST /submissions/:id/approve, POST /submissions/:id/reject

  internal/router/router.go                Updated — submissions module wired
```

Key behaviours:
- Eligibility: campaign must be active + not expired + has slots; social account must match platform and meet min followers/engagement/influence
- Fraud: Redis rate limit (3/hour); duplicate repost URL caught by DB unique index; low follower ratio creates FraudFlag + -30 trust points async
- Payout formula at submission time: influenceMultiplier = 0.5 + (influenceScore/100); finalAmount = baseRepostRate × multiplier; platformFee = 20%; promoterEarning stored on submission
- Approval: business pendingBalance -= finalAmount; campaign remainingBudget -= finalAmount; promoter pendingBalance += promoterEarning; payoutReleasedAt = now + 48h; promoter trustScore += 5
- Rejection: campaign currentParticipants--; promoter trustScore -= 15; auto-suspend if < 20
- Screenshot upload: multipart POST to /submissions/upload, saved to {UploadDir}/screenshots/, 10 MB limit

## Phase 3 — Files Created

```
apps/api/
  internal/modules/users/dto.go        UpdateProfileRequest, ConnectSocialAccountRequest, UserResponse, SocialAccountResponse, InfluenceScoreResponse + mappers
  internal/modules/users/service.go    getMe, updateProfile, connectSocialAccount, deleteSocialAccount, getInfluenceScore + influence score sub-components
  internal/modules/users/handler.go    HTTP handlers for all user routes
  internal/modules/users/routes.go     GET /users/me, PATCH /users/me, GET /users/influence-score, POST /users/social-accounts, DELETE /users/social-accounts/:id

  internal/modules/campaigns/dto.go    CreateCampaignRequest, UpdateCampaignRequest, CampaignResponse, CampaignListQuery, CampaignListMeta + mapper
  internal/modules/campaigns/service.go createCampaign (locks wallet), getCampaigns (marketplace), getCampaign, getMyCampaigns, updateCampaign, deleteCampaign (refunds wallet)
  internal/modules/campaigns/handler.go HTTP handlers for all campaign routes
  internal/modules/campaigns/routes.go  GET /campaigns, POST /campaigns, GET /campaigns/my, GET /campaigns/:id, PATCH /campaigns/:id, DELETE /campaigns/:id

  internal/router/router.go             Updated — users + campaigns modules wired
```

Also fixed in this session:
- All models (`user.go`, `campaign.go`, `social_account.go`, `wallet.go`, `submission.go`, `notification.go`, `fraud.go`) migrated from `primitive.ObjectID` → `bson.ObjectID` (mongo-driver v2 dropped the `bson/primitive` sub-package)
- `auth/service.go` broken InsertedID cast removed (re-fetch already sets the ID)
- `go.sum` generated via `go mod tidy`

## Phase 2 — Files Created

```
apps/api/
  internal/utils/jwt.go               GenerateAccessToken, GenerateRefreshToken, Validate*
  internal/utils/hash.go              HashPassword, CheckPassword (bcrypt 12 rounds)
  internal/utils/token.go             GenerateSecureToken (crypto/rand hex)
  internal/services/email.go          SendVerificationEmail, SendPasswordResetEmail (gomail)
  internal/database/indexes.go        All MongoDB indexes, called at startup
  internal/modules/auth/dto.go        RegisterRequest, LoginRequest, AuthResponse, etc.
  internal/modules/auth/service.go    register, login, logout, refresh, verifyEmail, forgotPassword, resetPassword
  internal/modules/auth/handler.go    HTTP handlers for all auth endpoints
  internal/modules/auth/routes.go     Route registration
  internal/middleware/auth.go         RequireAuth() — JWT validation, sets userID+role in context
  internal/middleware/roles.go        RequireRole(...roles) — role-based access guard
  internal/router/router.go           Updated — auth routes wired
  cmd/server/main.go                  Updated — CreateIndexes() called at startup
```

## Phase 1 — Files Created

```
apps/api/
  cmd/server/main.go
  internal/config/config.go
  internal/database/mongodb.go
  internal/database/redis.go
  internal/models/user.go
  internal/models/social_account.go
  internal/models/campaign.go
  internal/models/submission.go
  internal/models/wallet.go          (also contains Transaction + Withdrawal)
  internal/models/notification.go
  internal/models/fraud.go
  internal/router/router.go
  internal/utils/response.go
  Dockerfile
  go.mod
  .env.example

apps/web/
  package.json
  tsconfig.json
  tailwind.config.ts
  postcss.config.js
  next.config.ts
  src/app/layout.tsx
  src/app/page.tsx
  src/app/globals.css
  src/lib/axios.ts
  src/lib/utils.ts
  src/types/index.ts
  Dockerfile
  .env.example

docker-compose.yml
```

---

## Known Issues / Decisions Pending

- `go.sum` is committed — `go mod tidy` already run, no action needed
- `apps/web/node_modules` not installed — run `npm install` inside `apps/web/`
- Copy `.env.example` → `.env` in both apps before running docker-compose

---

## Key Decisions Already Made

| Decision | Choice | Reason |
|---|---|---|
| Backend language | Node.js/Express | Better fit for MongoDB + Socket.IO + Stripe, faster iteration |
| Component library | shadcn/ui | Fastest path to premium design (Stripe/Linear aesthetic) |
| Proof verification | URL + screenshot, manual admin review | Instagram/Twitter APIs don't support programmatic repost verification |
| Payout hold period | 48 hours | Standard fraud buffer |
| Stripe payout model | Stripe Connect Express | Only scalable option for paying out to individual users |
| Commission timing | Deducted at point of approval | Simplest to audit and reverse |
| MVP platforms | Instagram + Twitter/X only | Scope control; architecture supports adding more later |

---

## README Update Rule

**Update `README.md` at the end of every session** alongside CLAUDE.md.

Specifically, after each phase update:
- The "What's Built So Far" phase table (mark completed phases ✅)
- The "Project Structure" section if new folders/files were added
- The "Prerequisites" or "First-Time Setup" sections if new tools or setup steps are required
- The "Common Issues" section if any notable gotchas were found during the build
- The "Environment Variables Reference" if new env vars were added to `.env.example`

Never remove setup steps — only add or clarify them.

---

## How to Continue in a New Session

1. Read this file
2. Check the Phase table above for current status
3. Read any files already created in `apps/api/src/` and `apps/web/`
4. Continue from "Next action" above
5. Update this file before ending the session
