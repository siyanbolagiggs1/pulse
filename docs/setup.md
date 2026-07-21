# Setup

Two ways to run Pulse locally: Docker Compose (fastest, matches production
packaging) or a native dev container / manual setup (faster iteration, no
container rebuilds).

## Option 1 — Docker Compose

```bash
cp .env.example .env   # fill in real values where needed
make up                 # or: docker compose up --build -d
```

- API: http://localhost:5000
- Web: http://localhost:3000
- `make logs` to follow logs, `make down` to stop, `make help` for the full list.

## Option 2 — Native (VS Code Dev Container or manual)

Requires Go 1.22+, Node 20+, MongoDB 7.0, and Redis running locally.

```bash
# API
cp api/common/.env.example api/common/.env   # then edit
cd api/common && go mod download && go run ./cmd/server

# Web (separate terminal)
cp web/.env.example web/.env.local           # then edit
cd web && npm install && npm run dev
```

Opening this repo in VS Code with the Dev Containers extension runs
`.devcontainer/scripts/setup.sh` automatically, which installs MongoDB/Redis
and writes working dev `.env` files for both services. Use the "Start API" /
"Start Web" / "Start All" tasks (`Ctrl+Shift+P` → *Run Task*) to launch both.

## Required environment variables

See `api/common/.env.example` and `web/.env.example` for the full list.
The essentials to get a working local instance:

| Variable | Where | Purpose |
|---|---|---|
| `MONGODB_URI`, `DB_NAME` | API | MongoDB connection |
| `REDIS_URL` | API | Rate limiting |
| `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET` | API | Auth token signing |
| `PAYSTACK_SECRET_KEY` | API | Leave empty for dev mode (wallet top-ups credit directly, no real payment) |
| `NEXT_PUBLIC_API_URL` | Web | Where the frontend sends API requests |

Full deployment instructions (Railway + Vercel, or self-hosted VPS with
Docker + Caddy) are in the root [README.md](../README.md).
