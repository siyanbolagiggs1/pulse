#!/usr/bin/env bash
set -e

echo "==> Installing MongoDB 7.0..."
curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc \
  | sudo gpg --dearmor -o /usr/share/keyrings/mongodb-server-7.0.gpg
echo "deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] \
  https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/7.0 multiverse" \
  | sudo tee /etc/apt/sources.list.d/mongodb-org-7.0.list

echo "==> Installing Redis..."
sudo apt-get update -qq
sudo apt-get install -y mongodb-org redis-server

# Ensure mongod data directory exists
sudo mkdir -p /data/db
sudo chown -R vscode:vscode /data/db

echo "==> Setting up environment files..."

# API .env — uses localhost (no Docker service names or auth)
cat > apps/api/.env <<'EOF'
PORT=5000
NODE_ENV=development
MONGODB_URI=mongodb://localhost:27017/pulse
DB_NAME=pulse
REDIS_URL=redis://localhost:6379
JWT_ACCESS_SECRET=dev-access-secret-replace-before-prod-deploy
JWT_REFRESH_SECRET=dev-refresh-secret-replace-before-prod-deploy
JWT_ACCESS_EXPIRY_MINUTES=15
JWT_REFRESH_EXPIRY_DAYS=7
STRIPE_SECRET_KEY=
STRIPE_WEBHOOK_SECRET=
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
SMTP_FROM=noreply@pulse.app
CLIENT_URL=*
UPLOAD_DIR=./uploads
PLATFORM_COMMISSION_RATE=0.20
EOF

# Web .env.local — uses relative /api so Next.js proxy handles routing
cat > apps/web/.env.local <<'EOF'
NEXT_PUBLIC_API_URL=/api
NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_placeholder
EOF

echo "==> Installing Go dependencies..."
cd apps/api && go mod download && cd ../..

echo "==> Installing Node dependencies..."
cd apps/web && npm install && cd ../..

echo "==> Setup complete. Use the VS Code tasks (Ctrl+Shift+P → Run Task) to start the app."
