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
| 5 | Wallet + Paystack | ✅ Complete |
| 6 | Admin panel | ✅ Complete |
| 7 | Influence scoring + fraud detection | ✅ Complete |
| 8 | Real-time notifications (SSE) | ✅ Complete |
| 9 | Frontend pages + dashboards | ✅ Complete |
| 10 | Production deploy config (CI/CD, Railway, Vercel) | ✅ Complete |
| 11 | Polish + missing UX (profile, social accounts, campaign edit, pagination, mobile nav) | ✅ Complete |
| 12 | Real Paystack payouts (bank account verification, Transfer API, webhook completion) | ✅ Complete |

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
| `PAYSTACK_SECRET_KEY` | Paystack Dashboard → Settings → API Keys |
| `PAYSTACK_PUBLIC_KEY` | Paystack Dashboard → Settings → API Keys |
| `PAYSTACK_CURRENCY` | e.g. `NGN`, `USD`, `GHS` |
| `SMTP_USER` | Your Gmail address |
| `SMTP_PASS` | Gmail → App Passwords (not your main password) |

Open `apps/web/.env.local` and set:

| Variable | Where to get it |
|---|---|
| `NEXT_PUBLIC_API_URL` | `http://localhost:5000/api` for local dev |

> For local development, Paystack test keys (`sk_test_...` / `pk_test_...`) are fine.

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

Two options: **managed** (Railway + Vercel, easiest, free) or **self-hosted** (VPS with Docker + Caddy, full control).

---

### Option 1 — Railway (API) + Vercel (Web) — Recommended

#### API → Railway

1. Go to [railway.app](https://railway.app) → **New Project → Deploy from GitHub repo**
2. Select your repo and set **Root Directory** to `apps/api`
3. Railway auto-detects the Dockerfile and builds it
4. Under **Variables**, add all required secrets (see Environment Variables Reference below)
5. Note the public URL Railway assigns — e.g. `https://pulse-api-production.up.railway.app`

> **File uploads note:** Railway's free tier has no persistent volumes. Uploaded screenshots are lost on redeploy. Use [Cloudinary](https://cloudinary.com) (free 25GB) for persistent file storage when you're ready.

#### Web → Vercel

```bash
# Install Vercel CLI: npm i -g vercel
cd apps/web

# First time only — links to your Vercel account/org
vercel link

# Set environment variables in Vercel dashboard:
#   NEXT_PUBLIC_API_URL = https://your-railway-url.up.railway.app/api

# Deploy to production
vercel --prod
```

Or connect the GitHub repo in the Vercel dashboard (set "Root Directory" to `apps/web`) and every push to `main` deploys automatically.

#### Automated CI/CD

Add these secrets to GitHub → Settings → Secrets and variables → Actions:

| Secret | How to get it |
|---|---|
| `RAILWAY_TOKEN` | Railway dashboard → Account Settings → Tokens |
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

# Start everything (Caddy handles HTTPS automatically)
docker-compose -f docker-compose.prod.yml --env-file apps/api/.env.prod up -d --build
```

Caddy provisions a Let's Encrypt certificate automatically on first startup. The stack is:
- **Port 80/443** → Caddy (public)
- `/api/*` → Go API container (internal only)
- Everything else → Next.js container (internal only)
- MongoDB and Redis are not exposed outside the Docker network

---

## Paystack Local Webhooks (required for wallet top-ups)

When testing Paystack payments locally you need to expose your local API to the internet so Paystack can send webhook events to it.

```bash
# Option A — ngrok (easiest)
ngrok http 5000

# Copy the https URL it gives you, e.g. https://abc123.ngrok.io
# In Paystack Dashboard → Settings → Webhooks, add:
#   https://abc123.ngrok.io/api/wallet/topup/webhook

# Option B — use Paystack's test mode
# Paystack test mode payments trigger real webhook calls to your registered URL.
# Just make sure PAYSTACK_SECRET_KEY is set to your test key (sk_test_...).
```

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
│   │   │   ├── services/     Influence scoring, fraud, email, paystack
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
│       │   │       ├── wallet/       Balance, top-up, withdraw
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
| `PAYSTACK_SECRET_KEY` | — | Paystack secret key (`sk_test_...` or `sk_live_...`) |
| `PAYSTACK_PUBLIC_KEY` | — | Paystack public key (`pk_test_...` or `pk_live_...`) |
| `PAYSTACK_CURRENCY` | `NGN` | Transaction currency (NGN, USD, GHS, ZAR, KES) |
| `SMTP_HOST` | `smtp.gmail.com` | SMTP server |
| `SMTP_PORT` | `587` | SMTP port |
| `SMTP_USER` | — | SMTP login |
| `SMTP_PASS` | — | SMTP password / app password |
| `SMTP_FROM` | `noreply@pulse.app` | Sender email address |
| `CLIENT_URL` | `http://localhost:3000` | Frontend URL (used in email links + CORS) |
| `UPLOAD_DIR` | `./uploads` | Where proof screenshots are saved |
| `PLATFORM_COMMISSION_RATE` | `0.20` | Platform cut (0.20 = 20%) |
| `MONGO_ROOT_USER` | `admin` | MongoDB root user (Docker only) |
| `MONGO_ROOT_PASS` | `secret` | MongoDB root password (Docker only) |

### Frontend (`apps/web/.env.local`)

| Variable | Description |
|---|---|
| `NEXT_PUBLIC_API_URL` | API base URL (e.g. `http://localhost:5000/api`) |

---

## API Endpoints

### Auth

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

### Users

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| GET | `/api/users/me` | Bearer | Get own profile + social accounts |
| PATCH | `/api/users/me` | Bearer | Update name / avatar |
| GET | `/api/users/influence-score` | Bearer | Influence score breakdown per social account |
| POST | `/api/users/social-accounts` | Bearer | Connect a social account (Instagram or Twitter) |
| DELETE | `/api/users/social-accounts/:id` | Bearer | Remove a social account |
| GET | `/api/users/banks` | Bearer | List banks for the configured Paystack currency |
| POST | `/api/users/bank-account` | Bearer | Verify + save a payout bank account (resolves account name via Paystack) |

### Campaigns

| Method | Endpoint | Auth | Role | Description |
|---|---|---|---|---|
| GET | `/api/campaigns` | Bearer | any | Browse active campaigns (marketplace) |
| POST | `/api/campaigns` | Bearer | business | Create campaign (locks budget from wallet) |
| GET | `/api/campaigns/my` | Bearer | business | List own campaigns |
| GET | `/api/campaigns/:id` | Bearer | any | Get single campaign |
| PATCH | `/api/campaigns/:id` | Bearer | business | Update campaign |
| DELETE | `/api/campaigns/:id` | Bearer | business | Delete campaign (refunds remaining budget) |

### Submissions

| Method | Endpoint | Auth | Role | Description |
|---|---|---|---|---|
| POST | `/api/submissions/upload` | Bearer | promoter | Upload proof screenshot → returns URL |
| POST | `/api/submissions` | Bearer | promoter | Submit proof for a campaign |
| GET | `/api/submissions` | Bearer | any | List submissions (scoped by role) |
| GET | `/api/submissions/:id` | Bearer | any | Get single submission (scoped by role) |
| POST | `/api/submissions/:id/approve` | Bearer | admin | Approve → credits promoter pending balance |
| POST | `/api/submissions/:id/reject` | Bearer | admin | Reject with reason → adjusts trust score |

### Wallet

| Method | Endpoint | Auth | Role | Description |
|---|---|---|---|---|
| GET | `/api/wallet` | Bearer | any | Balance + last 10 transactions (triggers 48h release) |
| GET | `/api/wallet/transactions` | Bearer | any | Paginated transaction history |
| POST | `/api/wallet/topup` | Bearer | business | Initiate Paystack payment → returns `authorizationUrl` |
| GET | `/api/wallet/topup/verify` | Bearer | any | Verify payment by reference after Paystack redirect |
| POST | `/api/wallet/topup/webhook` | — | — | Paystack webhook: credits wallet on `charge.success`; completes/refunds withdrawals on `transfer.success`/`transfer.failed`/`transfer.reversed` |
| POST | `/api/wallet/withdraw` | Bearer | promoter | Request withdrawal (requires a saved bank account; creates pending record) |
| GET | `/api/wallet/withdrawals` | Bearer | promoter | Paginated withdrawal history |

### Admin

| Method | Endpoint | Auth | Role | Description |
|---|---|---|---|---|
| GET | `/api/admin/stats` | Bearer | admin | Platform stats (users, campaigns, submissions, financials) |
| GET | `/api/admin/users` | Bearer | admin | List users with filters |
| GET | `/api/admin/users/:id` | Bearer | admin | Get single user |
| POST | `/api/admin/users/:id/suspend` | Bearer | admin | Suspend user with reason |
| POST | `/api/admin/users/:id/unsuspend` | Bearer | admin | Reinstate user (resets trust score to 50) |
| GET | `/api/admin/fraud-flags` | Bearer | admin | List fraud flags |
| POST | `/api/admin/fraud-flags/:id/resolve` | Bearer | admin | Mark fraud flag resolved |
| GET | `/api/admin/withdrawals` | Bearer | admin | List withdrawals with filters |
| POST | `/api/admin/withdrawals/:id/approve` | Bearer | admin | Approve withdrawal → creates/reuses a Paystack transfer recipient and initiates a real Transfer |
| POST | `/api/admin/withdrawals/:id/reject` | Bearer | admin | Reject → refunds balance to promoter wallet |

### Notifications

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

**Paystack webhook not receiving events**
Make sure your webhook URL is registered in the Paystack dashboard (Settings → Webhooks) and that `PAYSTACK_SECRET_KEY` is set correctly.

**Withdrawal approval fails with an OTP error**
Admin-approving a withdrawal calls Paystack's Transfers API directly, with no human available to answer an SMS OTP prompt. Go to the Paystack dashboard → Settings → Preferences and disable OTP for transfers, then retry the approval — nothing was charged or moved on the failed attempt, so it's safe to just retry.

**Emails not sending**
For Gmail, you must use an [App Password](https://myaccount.google.com/apppasswords), not your main Gmail password. Enable 2FA first, then generate the app password and set it as `SMTP_PASS`.

**CORS errors in production**
Make sure `CLIENT_URL` in `apps/api/.env.prod` exactly matches the frontend origin (including scheme, no trailing slash): `https://your-app.vercel.app`. The API uses this value as the only allowed CORS origin.

**"Email already registered" on fresh database**
MongoDB unique index on `users.email` is enforced. Drop the collection or use a different email.

**Login returns "please verify your email"**
Check your inbox (and spam) for the verification email. For local dev without real SMTP, temporarily set `isEmailVerified: true` directly in MongoDB using a GUI like MongoDB Compass.

---

## Going Live — Quick Start

Everything below is free until you hit scale.

### Services you need to sign up for

| Service | Purpose | Cost |
|---|---|---|
| [railway.app](https://railway.app) | Host the Go API | ~$5 credit/month free |
| [vercel.com](https://vercel.com) | Host the Next.js frontend | Free (hobby tier) |
| [MongoDB Atlas](https://cloud.mongodb.com) | Database | Free (512 MB cluster) |
| [Upstash](https://upstash.com) | Redis | Free (10k req/day) |
| [Paystack](https://paystack.com) | Payments | Free (% per transaction) |

---

### Step 1 — MongoDB Atlas

1. Create a free **M0** cluster
2. Create a database user with a password
3. Under Network Access, add `0.0.0.0/0` (Railway uses dynamic IPs)
4. Click **Connect → Drivers** and copy your connection string:
   `mongodb+srv://user:pass@cluster.mongodb.net/pulse?retryWrites=true&w=majority`

---

### Step 2 — Upstash Redis

1. Create a Redis database — pick the region closest to your Railway region
2. Copy the **Redis URL** — it looks like `rediss://default:password@host:port`

---

### Step 3 — Deploy the API to Railway

1. Go to [railway.app](https://railway.app) → **New Project → Deploy from GitHub repo**
2. Select your repo, set **Root Directory** to `apps/api`
3. Railway detects the Dockerfile and builds automatically
4. Under **Variables**, add all secrets:

| Variable | Value |
|---|---|
| `MONGODB_URI` | Your Atlas connection string |
| `REDIS_URL` | Your Upstash Redis URL |
| `JWT_ACCESS_SECRET` | Any random 32+ char string |
| `JWT_REFRESH_SECRET` | Different random 32+ char string |
| `PAYSTACK_SECRET_KEY` | `sk_live_...` from Paystack dashboard |
| `PAYSTACK_PUBLIC_KEY` | `pk_live_...` from Paystack dashboard |
| `PAYSTACK_CURRENCY` | e.g. `NGN` |
| `SMTP_HOST` | `smtp.gmail.com` |
| `SMTP_PORT` | `587` |
| `SMTP_USER` | Your Gmail address |
| `SMTP_PASS` | Your Gmail App Password |
| `SMTP_FROM` | `noreply@yourdomain.com` |
| `CLIENT_URL` | `https://your-app.vercel.app` (update after Step 4) |
| `PORT` | `5000` |
| `DB_NAME` | `pulse` |
| `PLATFORM_COMMISSION_RATE` | `0.20` |

5. Under **Settings → Networking**, generate a public domain
6. Your API is live at `https://your-service.up.railway.app`

---

### Step 4 — Deploy the frontend to Vercel

```bash
npm install -g vercel
cd apps/web
vercel
```

Follow the prompts. When asked for environment variables, add:

```
NEXT_PUBLIC_API_URL = https://your-service.up.railway.app/api
```

Once Vercel gives you your URL (e.g. `https://pulse-xyz.vercel.app`), go back to Railway → Variables and update:

```
CLIENT_URL = https://pulse-xyz.vercel.app
```

---

### Step 5 — Set up Paystack webhook

In your Paystack dashboard → **Settings → Webhooks**:

- Add URL: `https://your-service.up.railway.app/api/wallet/topup/webhook`
- Events: `charge.success`

---

### Step 6 — Create your admin account

1. Register an account on your live site
2. In MongoDB Atlas → Browse Collections → `users`, find your document and set `role` to `"admin"`

---

### Step 7 — Wire up auto-deploy from GitHub (optional)

Every push to `main` will automatically redeploy both apps.

Add these secrets to your GitHub repo → **Settings → Secrets → Actions**:

| Secret | How to get it |
|---|---|
| `RAILWAY_TOKEN` | Railway dashboard → Account Settings → Tokens |
| `VERCEL_TOKEN` | Vercel dashboard → Account Settings → Tokens |
| `VERCEL_ORG_ID` | Run `cat apps/web/.vercel/project.json` after `vercel link` |
| `VERCEL_PROJECT_ID` | Same file as above |

---

### Gmail App Password (for email to work)

Regular Gmail passwords don't work with SMTP. You need an App Password:

1. Go to your Google account → **Security**
2. Enable **2-Step Verification** if not already on
3. Go to **App Passwords** → generate one for "Mail"
4. Use that 16-character password as `SMTP_PASS`

---

*This README is updated continuously as the app is built.*









<!-- env -->
<!-- env -->
<!-- env -->
<!-- env -->
<!-- env -->
┌───────────────────────────┬──────────────────────────────────┐
  │         Variable          │              Value               │
  ├───────────────────────────┼──────────────────────────────────┤
  │ NODE_ENV                  │ production ← change from         │
  │                           │ development                      │
  ├───────────────────────────┼──────────────────────────────────┤
  │ PORT                      │ 5000                             │
  ├───────────────────────────┼──────────────────────────────────┤
  │ DB_NAME                   │ pulse                            │
  ├───────────────────────────┼──────────────────────────────────┤
  │ JWT_ACCESS_EXPIRY_MINUTES │ 15                               │
  ├───────────────────────────┼──────────────────────────────────┤
  │ JWT_REFRESH_EXPIRY_DAYS   │ 7                                │
  ├───────────────────────────┼──────────────────────────────────┤
  │ GOOGLE_CLIENT_ID          │ your existing value              │
  ├───────────────────────────┼──────────────────────────────────┤
  │ TWITTER_BEARER_TOKEN      │ your existing value              │
  ├───────────────────────────┼──────────────────────────────────┤
  │ PAYSTACK_SECRET_KEY       │ your existing sk_test_... value  │
  ├───────────────────────────┼──────────────────────────────────┤
  │ PAYSTACK_PUBLIC_KEY       │ your existing pk_test_... value  │
  ├───────────────────────────┼──────────────────────────────────┤
  │ PAYSTACK_CURRENCY         │ NGN                              │
  ├───────────────────────────┼──────────────────────────────────┤
  │ SMTP_HOST                 │ smtp.gmail.com                   │
  ├───────────────────────────┼──────────────────────────────────┤
  │ SMTP_PORT                 │ 587                              │
  ├───────────────────────────┼──────────────────────────────────┤
  │ SMTP_USER                 │ your Gmail                       │
  ├───────────────────────────┼──────────────────────────────────┤
  │ SMTP_PASS                 │ your existing app password       │
  ├───────────────────────────┼──────────────────────────────────┤
  │ SMTP_FROM                 │ your Gmail                       │
  ├───────────────────────────┼──────────────────────────────────┤
  │ PLATFORM_COMMISSION_RATE  │ 0.20                             │
  ├───────────────────────────┼──────────────────────────────────┤
  │ UPLOAD_DIR                │ /app/uploads ← change from       │
  │                           │ ./uploads                        │
  └───────────────────────────┴──────────────────────────────────┘

  These 4 need new values — localhost won't work on Railway:

  ┌────────────────────┬─────────────────────────────────────────┐
  │      Variable      │               What to set               │
  ├────────────────────┼─────────────────────────────────────────┤
  │ MONGODB_URI        │ Your Atlas connection string (sign up   │
  │                    │ at mongodb.com/atlas)                   │
  ├────────────────────┼─────────────────────────────────────────┤
  │ REDIS_URL          │ Your Upstash Redis URL (sign up at      │
  │                    │ upstash.com)                            │
  ├────────────────────┼─────────────────────────────────────────┤
  │ JWT_ACCESS_SECRET  │ Generate a strong one — run openssl     │
  │                    │ rand -hex 32 in your terminal           │
  ├────────────────────┼─────────────────────────────────────────┤
  │ JWT_REFRESH_SECRET │ Same — run again for a different value  │
  └────────────────────┴─────────────────────────────────────────┘

  Set last (after Vercel deploy gives you a URL):

  ┌────────────┬─────────────────────────────┐
  │  Variable  │            Value            │
  ├────────────┼─────────────────────────────┤
  │ CLIENT_URL │ https://your-app.vercel.app │
