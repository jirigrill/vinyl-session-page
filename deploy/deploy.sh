#!/usr/bin/env bash
# Build and deploy vinyl to the droplet. Idempotent; safe to re-run.
# Usage: deploy/deploy.sh [ssh-target]   (default root@142.93.99.205)
set -euo pipefail

HOST="${1:-root@142.93.99.205}"
cd "$(dirname "$0")/.."

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /tmp/vinyl ./cmd/vinyl

scp -q /tmp/vinyl "$HOST":/tmp/vinyl.new
scp -q deploy/vinyl.service deploy/Caddyfile.vinyl deploy/vinyl.env.example "$HOST":/tmp/

ssh "$HOST" 'set -euo pipefail
id vinyl >/dev/null 2>&1 || useradd --system --no-create-home --shell /usr/sbin/nologin vinyl
install -d -m 755 /opt/vinyl
install -m 755 /tmp/vinyl.new /opt/vinyl/vinyl
install -d -m 700 /etc/vinyl
[ -f /etc/vinyl/vinyl.env ] || { install -m 600 /tmp/vinyl.env.example /etc/vinyl/vinyl.env; echo "!! edit /etc/vinyl/vinyl.env, then: systemctl restart vinyl"; }
install -m 644 /tmp/vinyl.service /etc/systemd/system/vinyl.service
systemctl daemon-reload
systemctl enable --now vinyl
systemctl restart vinyl

grep -q "^vinyl.nofiat.me" /etc/caddy/Caddyfile || { printf "\n" >> /etc/caddy/Caddyfile; cat /tmp/Caddyfile.vinyl >> /etc/caddy/Caddyfile; systemctl reload caddy; }

rm -f /tmp/vinyl.new /tmp/vinyl.service /tmp/Caddyfile.vinyl /tmp/vinyl.env.example
sleep 1; systemctl is-active vinyl'
