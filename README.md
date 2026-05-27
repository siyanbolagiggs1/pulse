# Pulse

Community-powered social promotion marketplace.
Businesses run repost campaigns. Promoters earn money sharing them.

---

## What's Built So Far

| Phase | Scope | Status |
|---|---|---|
| 1 | Scaffold, Docker, models, base server | ✅ Complete |
| 2 | Auth system | ✅ Complete |
| 3 | Campaigns (business) | ✅ Complete |
| 4 | Marketplace + submissions (promoter) | ✅ Complete |
| 5 | Wallet + Stripe | ✅ Complete |
| 6 | Admin panel | ✅ Complete |
| 7 | Influence scoring + fraud detection | ✅ Complete |
| 8 | Real-time notifications (SSE) | ✅ Complete |
| 9 | Frontend pages + dashboards | ✅ Complete |

---

## Quickstart — GitHub Codespaces (no local setup needed)

The fastest way to run Pulse. Everything installs automatically in the cloud.

1. Push this repo to GitHub (see below if you haven't yet)
2. On the repo page → **Code → Codespaces → Create codespace on main**
3. Wait ~3 minutes for the container to build (Go, Node, MongoDB, Redis all install automatically)
4. When VS Code loads, press **Ctrl+Shift+P** → **Tasks: Run Task** → **Start All**
5. VS Code will prompt to open the forwarded port — click **Open in Browser**

The app opens at the Codespace's port-3000 URL. The API runs on port 5000 (proxied through Next.js — no CORS issues).

### First-time: push to GitHub

```bash
cd C:\Users\HP\Desktop\dev\pulse

git init
git add .
git commit -m "Initial commit"

# Create a repo on github.com, then:
git remote add origin https://github.com/YOUR_USERNAME/pulse.git
git push -u origin main
```

### Create an admin user

The database starts empty. Register a user at `/register`, then in the Codespace terminal promote it to admin:

```bash
# In the Codespace terminal
mongosh pulse --eval '
  db.users.updateOne(
    { email: "your@email.com" },
    { $set: { role: "admin" } }
  )
'
```

---

## Prerequisites

Make sure these are installed on your machine before anything else:

| Tool | Version | Install |
|---|---|---|
| Go | 1.22+ | https://go.dev/dl |
| Node.js | 20+ | https://nodejs.org |
| Docker Desktop | latest | https://www.docker.com/products/docker-desktop |
| Docker Compose | v2 (bundled with Docker Desktop) | — |

---

## First-Time Setup

Do these steps once after cloning the repo.

### 1. Install Go dependencies

```bash
cd apps/api
go mod tidy
```

This generates `go.sum`. You must do this before building the API container.

### 2. Install frontend dependencies

```bash
cd apps/web
npm install
```

### 3. Create environment files

```bash
# API
cp apps/api/.env.example apps/api/.env

# Frontend
cp apps/web/.env.example apps/web/.env.local
```

### 4. Fill in your secrets

Open `apps/api/.env` and set real values for:

| Variable | Where to get it |
|---|---|
| `JWT_ACCESS_SECRET` | Any random 32+ char string |
| `JWT_REFRESH_SECRET` | Any random 32+ char string (different from above) |
| `STRIPE_SECRET_KEY` | Stripe Dashboard → Developers → API Keys |
| `STRIPE_WEBHOOK_SECRET` | Stripe Dashboard → Webhooks (after running stripe CLI) |
| `SMTP_USER` | Your Gmail address |
| `SMTP_PASS` | Gmail → App Passwords (not your main password) |

Open `apps/web/.env.local` and set:

| Variable | Where to get it |
|---|---|
| `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY` | Stripe Dashboard → Developers → API Keys |

> For local development, Stripe test keys (`sk_test_...` / `pk_test_...`) are fine.

---

## Running the App

### Option A — Docker (recommended)

Starts everything: MongoDB, Redis, API, and frontend.

```bash
docker-compose up --build
```

| Service | URL |
|---|---|
| Frontend | http://localhost:3000 |
| API | http://localhost:5000 |
| API health check | http://localhost:5000/health |
| MongoDB | localhost:27017 |
| Redis | localhost:6379 |

To stop:
```bash
docker-compose down
```

To stop and wipe all data (mongo + redis volumes):
```bash
docker-compose down -v
```

---

### Option B — Local (without Docker)

Run MongoDB and Redis separately (or use Docker just for those):

```bash
# Start only the databases
docker-compose up mongodb redis
```

Then in separate terminals:

```bash
# Terminal 1 — API
cd apps/api
go run ./cmd/server

# Terminal 2 — Frontend
cd apps/web
npm run dev
```

---

## Production Deployment

Two options: **managed** (Fly.io + Vercel, easiest) or **self-hosted** (VPS with Docker + Caddy, full control).

---

### Option 1 — Fly.io (API) + Vercel (Web) — Recommended

#### API → Fly.io

```bash
# Install flyctl: https://fly.io/docs/hands-on/install-flyctl/
cd apps/api

# First time only
flyctl launch --no-deploy          # reads fly.toml, creates the app

# Set all secrets (do NOT commit these)
flyctl secrets set \
  MONGODB_URI="mongodb+srv://..." \
  REDIS_URL="rediss://..." \
  JWT_ACCESS_SECRET="$(openssl rand -hex 32)" \
  JWT_REFRESH_SECRET="$(openssl rand -hex 32)" \
  STRIPE_SECRET_KEY="sk_live_..." \
  STRIPE_WEBHOOK_SECRET="whsec_..." \
  SMTP_HOST="smtp.sendgrid.net" \
  SMTP_PORT="587" \
  SMTP_USER="apikey" \
  SMTP_PASS="SG...." \
  SMTP_FROM="noreply@pulse.app" \
  CLIENT_URL="https://pulse.vercel.app"

# Deploy
flyctl deploy
```

The API will be live at `https://pulse-api.fly.dev`.

For file uploads, Fly.io creates a persistent volume (`pulse_uploads`) as defined in `fly.toml`.

#### Web → Vercel

```bash
# Install Vercel CLI: npm i -g vercel
cd apps/web

# First time only — links to your Vercel account/org
vercel link

# Set environment variables in Vercel dashboard:
#   NEXT_PUBLIC_API_URL = https://pulse-api.fly.dev/api
#   NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY = pk_live_...

# Deploy to production
vercel --prod
```

Or just connect the GitHub repo in the Vercel dashboard (set "Root Directory" to `apps/web`) and every push to `main` deploys automatically.

#### Automated CI/CD

Add these secrets to GitHub → Settings → Secrets and variables → Actions:

| Secret | How to get it |
|---|---|
| `FLY_API_TOKEN` | `flyctl tokens create deploy` |
| `VERCEL_TOKEN` | Vercel dashboard → Account → Tokens |
| `VERCEL_ORG_ID` | `cat apps/web/.vercel/project.json` after `vercel link` |
| `VERCEL_PROJECT_ID` | same file |

After that, every push to `main` triggers `.github/workflows/deploy.yml` which deploys both services in sequence.

---

### Option 2 — Self-Hosted VPS (Docker + Caddy)

#### Prerequisites
- A VPS with Docker and Docker Compose v2 installed
- A domain with an A record pointing to the server's IP
- Edit `Caddyfile` — replace `pulse.app` with your actual domain

#### Setup

```bash
# On the server
git clone https://github.com/yourorg/pulse.git
cd pulse

# Create production env files
cp apps/api/.env.example apps/api/.env.prod
cp apps/web/.env.example apps/web/.env.prod

# Edit apps/api/.env.prod — fill all real values, set NODE_ENV=production
# Edit apps/web/.env.prod — set NEXT_PUBLIC_API_URL=https://your-domain.com/api

# Generate a Redis password and add to .env.prod as REDIS_PASSWORD=...
# Add NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_live_... to .env.prod

# Start everything (Caddy handles HTTPS automatically)
docker-compose -f docker-compose.prod.yml --env-file apps/api/.env.prod up -d --build
```

Caddy provisions a Let's Encrypt certificate automatically on first startup. The stack is:
- **Port 80/443** → Caddy (public)
- `/api/*` → Go API container (internal only)
- Everything else → Next.js container (internal only)
- MongoDB and Redis are not exposed outside the Docker network

---

## Stripe Local Webhooks (required for wallet top-ups)

When testing Stripe payments locally you need the Stripe CLI to forward webhook events to your local API.

```bash
# Install Stripe CLI: https://stripe.com/docs/stripe-cli

stripe login
stripe listen --forward-to http://localhost:5000/api/wallet/topup/webhook
```

Copy the `whsec_...` key it prints and paste it into `apps/api/.env` as `STRIPE_WEBHOOK_SECRET`.

---

## Project Structure

```
pulse/
├── apps/
│   ├── api/                  Go/Gin REST API
│   │   ├── cmd/server/       Entry point (main.go)
│   │   ├── internal/
│   │   │   ├── config/       Env config loader
│   │   │   ├── database/     MongoDB + Redis connections
│   │   │   ├── middleware/   Auth, roles, rate limiting, upload
│   │   │   ├── models/       MongoDB document structs
│   │   │   ├── modules/      Feature modules (auth, campaigns, etc.)
│   │   │   ├── router/       Gin router setup
│   │   │   ├── services/     Influence scoring, fraud, email, payout
│   │   │   └── utils/        Response helpers
│   │   ├── Dockerfile
│   │   └── .env.example
│   │
│   └── web/                  Next.js 14 frontend
│       ├── src/
│       │   ├── app/
│       │   │   ├── (auth)/           Login, register, forgot/reset password
│       │   │   ├── verify-email/     Email verification handler
│       │   │   └── dashboard/
│       │   │       ├── campaigns/    Business campaign CRUD
│       │   │       ├── marketplace/  Promoter campaign browse + apply
│       │   │       ├── submissions/  Submission list (role-scoped)
│       │   │       ├── wallet/       Balance, top-up, withdraw, Stripe Connect
│       │   │       └── admin/        Stats, users, submissions, fraud, withdrawals
│       │   ├── components/
│       │   │   ├── layout/           Sidebar, header (with SSE notifications)
│       │   │   └── ui/               shadcn/ui components (Radix-based)
│       │   ├── hooks/                useSSE (fetch + ReadableStream SSE)
│       │   ├── lib/                  Axios client, API functions, utilities
│       │   ├── store/                Zustand auth store
│       │   └── types/                Shared TypeScript types
│       ├── Dockerfile
│       └── .env.example
│
├── docker-compose.yml
├── README.md
└── CLAUDE.md                 AI build briefing (not for humans)
```

---

## Environment Variables Reference

### API (`apps/api/.env`)

| Variable | Default | Description |
|---|---|---|
| `PORT` | `5000` | API server port |
| `NODE_ENV` | `development` | `development` or `production` |
| `MONGODB_URI` | `mongodb://admin:secret@mongodb:27017/pulse?authSource=admin` | MongoDB connection string |
| `DB_NAME` | `pulse` | MongoDB database name |
| `REDIS_URL` | `redis://redis:6379` | Redis connection string |
| `JWT_ACCESS_SECRET` | — | Secret for signing access tokens |
| `JWT_REFRESH_SECRET` | — | Secret for signing refresh tokens |
| `JWT_ACCESS_EXPIRY_MINUTES` | `15` | Access token lifespan |
| `JWT_REFRESH_EXPIRY_DAYS` | `7` | Refresh token lifespan |
| `STRIPE_SECRET_KEY` | — | Stripe secret key |
| `STRIPE_WEBHOOK_SECRET` | — | Stripe webhook signing secret |
| `SMTP_HOST` | `smtp.gmail.com` | SMTP server |
| `SMTP_PORT` | `587` | SMTP port |
| `SMTP_USER` | — | SMTP login |
| `SMTP_PASS` | — | SMTP password / app password |
| `SMTP_FROM` | `noreply@pulse.app` | Sender email address |
| `CLIENT_URL` | `http://localhost:3000` | Frontend URL (used in email links) |
| `UPLOAD_DIR` | `./uploads` | Where proof screenshots are saved |
| `PLATFORM_COMMISSION_RATE` | `0.20` | Platform cut (0.20 = 20%) |
| `MONGO_ROOT_USER` | `admin` | MongoDB root user (Docker only) |
| `MONGO_ROOT_PASS` | `secret` | MongoDB root password (Docker only) |

### Frontend (`apps/web/.env.local`)

| Variable | Description |
|---|---|
| `NEXT_PUBLIC_API_URL` | API base URL |
| `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY` | Stripe publishable key |

---

## API Endpoints

### Auth (Phase 2)

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| POST | `/api/auth/register` | — | Register as `business` or `promoter` |
| POST | `/api/auth/login` | — | Login, sets httpOnly refresh cookie |
| POST | `/api/auth/logout` | — | Clears refresh cookie, blacklists token |
| POST | `/api/auth/refresh` | cookie | Get new access token |
| GET | `/api/auth/verify-email/:token` | — | Verify email from link |
| POST | `/api/auth/forgot-password` | — | Send password reset email |
| POST | `/api/auth/reset-password/:token` | — | Set new password |
| GET | `/api/auth/me` | Bearer | Get current user |

**Token flow:**
- Access token (15 min) → returned in response body, stored in `sessionStorage`
- Refresh token (7 days) → stored in `httpOnly` cookie, auto-used by axios interceptor
- On logout or password reset → refresh token blacklisted in Redis

### Users (Phase 3)

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| GET | `/api/users/me` | Bearer | Get own profile + social accounts |
| PATCH | `/api/users/me` | Bearer | Update name / avatar |
| GET | `/api/users/influence-score` | Bearer | Influence score breakdown per social account |
| POST | `/api/users/social-accounts` | Bearer | Connect a social account (Instagram or Twitter) |
| DELETE | `/api/users/social-accounts/:id` | Bearer | Remove a social account |

### Campaigns (Phase 3)

| Method | Endpoint | Auth | Role | Description |
|---|---|---|---|---|
| GET | `/api/campaigns` | Bearer | any | Browse active campaigns (marketplace) |
| POST | `/api/campaigns` | Bearer | business | Create campaign (locks budget from wallet) |
| GET | `/api/campaigns/my` | Bearer | business | List own campaigns |
| GET | `/api/campaigns/:id` | Bearer | any | Get single campaign |
| PATCH | `/api/campaigns/:id` | Bearer | business | Update campaign |
| DELETE | `/api/campaigns/:id` | Bearer | business | Delete campaign (refunds remaining budget) |

### Submissions (Phase 4)

| Method | Endpoint | Auth | Role | Description |
|---|---|---|---|---|
| POST | `/api/submissions/upload` | Bearer | promoter | Upload proof screenshot → returns URL |
| POST | `/api/submissions` | Bearer | promoter | Submit proof for a campaign |
| GET | `/api/submissions` | Bearer | any | List submissions (scoped by role) |
| GET | `/api/submissions/:id` | Bearer | any | Get single submission (scoped by role) |
| POST | `/api/submissions/:id/approve` | Bearer | admin | Approve → credits promoter pending balance |
| POST | `/api/submissions/:id/reject` | Bearer | admin | Reject with reason → adjusts trust score |

### Wallet (Phase 5)

| Method | Endpoint | Auth | Role | Description |
|---|---|---|---|---|
| GET | `/api/wallet` | Bearer | any | Balance + last 10 transactions (triggers 48h release) |
| GET | `/api/wallet/transactions` | Bearer | any | Paginated transaction history |
| POST | `/api/wallet/topup` | Bearer | business | Create Stripe Payment Intent → returns `clientSecret` |
| POST | `/api/wallet/topup/webhook` | — | — | Stripe webhook: credits wallet on `payment_intent.succeeded` |
| POST | `/api/wallet/connect` | Bearer | promoter | Start Stripe Connect Express onboarding → returns URL |
| GET | `/api/wallet/connect/status` | Bearer | promoter | Sync + return Connect account status |
| POST | `/api/wallet/withdraw` | Bearer | promoter | Request withdrawal (creates pending record) |
| GET | `/api/wallet/withdrawals` | Bearer | promoter | Paginated withdrawal history |

### Admin (Phase 6)

| Method | Endpoint | Auth | Role | Description |
|---|---|---|---|---|
| GET | `/api/admin/stats` | Bearer | admin | Platform stats (users, campaigns, submissions, financials) |
| GET | `/api/admin/users` | Bearer | admin | List users with filters (role, suspended, search) |
| GET | `/api/admin/users/:id` | Bearer | admin | Get single user |
| POST | `/api/admin/users/:id/suspend` | Bearer | admin | Suspend user with reason |
| POST | `/api/admin/users/:id/unsuspend` | Bearer | admin | Reinstate user (resets trust score to 50) |
| GET | `/api/admin/fraud-flags` | Bearer | admin | List fraud flags with filters (userId, resolved) |
| POST | `/api/admin/fraud-flags/:id/resolve` | Bearer | admin | Mark fraud flag resolved |
| GET | `/api/admin/withdrawals` | Bearer | admin | List withdrawals with filters (userId, status) |
| POST | `/api/admin/withdrawals/:id/approve` | Bearer | admin | Approve → fires Stripe Transfer to promoter |
| POST | `/api/admin/withdrawals/:id/reject` | Bearer | admin | Reject → refunds balance to promoter wallet |

### Notifications (Phase 8)

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| GET | `/api/notifications/stream` | Bearer | SSE stream — push notifications in real-time |
| GET | `/api/notifications` | Bearer | Paginated notification list + unread count |
| POST | `/api/notifications/read-all` | Bearer | Mark all notifications as read |
| POST | `/api/notifications/:id/read` | Bearer | Mark a single notification as read |

**SSE event format:**
```
event: connected
data: {}

event: notification
data: {"id":"...","type":"submission_approved","title":"...","message":"...","isRead":false,"createdAt":"..."}

: heartbeat
```

Notification types: `submission_approved`, `submission_rejected`, `withdrawal_processed`, `wallet_topup`

---

## Common Issues

**`go mod tidy` fails**
Make sure Go 1.22+ is installed: `go version`

**Docker build fails on `go.sum not found`**
Run `go mod tidy` inside `apps/api/` first.

**Port already in use**
Change the host port in `docker-compose.yml` (left side of `ports:` mapping).

**MongoDB auth error**
Make sure `MONGO_ROOT_USER` and `MONGO_ROOT_PASS` in your `.env` match what's in `docker-compose.yml`.

**Stripe webhook not receiving events**
Make sure `stripe listen` is running and `STRIPE_WEBHOOK_SECRET` is set to the `whsec_...` value it printed.

**Emails not sending**
For Gmail, you must use an [App Password](https://myaccount.google.com/apppasswords), not your main Gmail password. Enable 2FA first, then generate the app password and set it as `SMTP_PASS`.

**CORS errors in production**
Make sure `CLIENT_URL` in `apps/api/.env.prod` exactly matches the frontend origin (including scheme, no trailing slash): `https://pulse.app`. The API uses this value as the only allowed CORS origin.

**"Email already registered" on fresh database**
MongoDB unique index on `users.email` is enforced. Drop the collection or use a different email.

**Login returns "please verify your email"**
Check your inbox (and spam) for the verification email. For local dev without real SMTP, temporarily set `isEmailVerified: true` directly in MongoDB using a GUI like MongoDB Compass.

---

*This README is updated continuously as the app is built.*
