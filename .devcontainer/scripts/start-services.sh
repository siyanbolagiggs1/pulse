#!/usr/bin/env bash
set -e

echo "==> Starting MongoDB..."
if ! pgrep mongod > /dev/null; then
  mongod --fork --logpath /tmp/mongod.log --dbpath /data/db
  echo "    mongod started."
else
  echo "    mongod already running."
fi

echo "==> Starting Redis..."
if ! pgrep redis-server > /dev/null; then
  redis-server --daemonize yes --logfile /tmp/redis.log
  echo "    redis-server started."
else
  echo "    redis-server already running."
fi

echo "==> Services ready."
echo ""
echo "    Start the API:     Ctrl+Shift+P → 'Tasks: Run Task' → 'Start API'"
echo "    Start the web app: Ctrl+Shift+P → 'Tasks: Run Task' → 'Start Web'"
echo "    Or run both at once:               'Start All'"
