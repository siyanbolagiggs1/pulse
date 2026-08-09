#!/usr/bin/env bash
# Hammers the Ingress (http://localhost/health) and prints which pod answered
# each request, colored per pod. This is the visual proof for the Loom
# recording that the load balancer is actually spreading traffic across
# pulse-api replicas — not just serving repeats of the same one pod.
#
# Run alongside bootstrap.sh's other watch commands:
#   bash k8s/watch-lb.sh
#
set -u

URL="http://localhost/health"
COLORS=(31 32 33 34 35 36 91 92 93 94 95 96)
declare -A pod_color
next_color=0

while true; do
  body=$(curl -s --max-time 2 "$URL")
  pod=$(echo "$body" | sed -n 's/.*"pod":"\([^"]*\)".*/\1/p')
  [ -z "$pod" ] && pod="(no response yet — is the Ingress up?)"

  if [ -z "${pod_color[$pod]:-}" ]; then
    pod_color[$pod]=${COLORS[$((next_color % ${#COLORS[@]}))]}
    next_color=$((next_color + 1))
  fi
  c=${pod_color[$pod]}

  printf "\033[%sm%s  %s\033[0m\n" "$c" "$(date +%H:%M:%S)" "$pod"
  sleep 0.3
done
