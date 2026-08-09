# Local HPA + load balancer demo (kind, inside Codespaces)

Self-contained kind cluster running `pulse-api` + mongo + redis, fronted by an
nginx Ingress Controller (the load balancer) and watched by a
HorizontalPodAutoscaler that scales `pulse-api` on CPU. No local Docker
needed — this runs entirely inside the Codespace via `docker-in-docker`.

## Setup

```bash
bash k8s/bootstrap.sh
```

Takes ~5-10 minutes on first run (image build + metrics-server install).
Re-running is fast — it reuses the existing cluster.

## Demo script

Open four terminals.

Terminal 1 — the HPA's decision-making, leave this running:

```bash
kubectl get hpa pulse-api -w
```

Terminal 2 — pods actually being created/terminated as the HPA acts:

```bash
kubectl get pods -w
```

Terminal 3 — the load balancer in action: every 0.3s it hits `pulse-api`
through the Ingress and prints which pod answered, colored per pod. This is
the payoff shot — while terminal 2 shows a new pod appear, this terminal
shows a new color join the rotation, proving the load balancer picked it up
and started sending it traffic:

```bash
bash k8s/watch-lb.sh
```

Terminal 4 — drive the traffic that makes the HPA scale:

```bash
# ramp up traffic
kubectl scale deployment/load-generator --replicas=6

# watch terminal 1: TARGETS climbs past 50%, REPLICAS goes 1 -> 2/3
# (metrics-server scrapes every 15s, HPA re-evaluates every 15s — expect
# a scale-up within ~30-60s of ramping up)
# watch terminal 3: a new pod color joins the rotation as each new
# replica comes up and the load balancer starts routing to it

# stop traffic
kubectl scale deployment/load-generator --replicas=0

# watch terminal 1: REPLICAS drops back to 1 after the stabilization
# window (60s here — see k8s/hpa.yaml, default is 300s in real clusters)
# watch terminal 3: colors drop back out as their pods terminate
```

Note: `load-generator` (terminal 4's traffic) hits the Service directly
in-cluster to drive CPU load — it's a separate path from terminal 3's
`watch-lb.sh`, which goes through the Ingress from outside the cluster like
a real client would. Both are hitting the same growing/shrinking pool of
pods, just to make two different things visible at once.

## Cleanup

```bash
kind delete cluster --name pulse
```

## Notes

- The load balancer is a real nginx Ingress Controller (`k8s/ingress.yaml`,
  installed by `bootstrap.sh`), not just the Service's built-in round-robin —
  it's the single entrypoint on `http://localhost` (hostPorts 80/443, mapped
  in via `k8s/kind-config.yaml` at cluster-creation time) and reverse-proxies
  each request to whichever `pulse-api` pod it picks next.
- `/health` returns a `pod` field sourced from the `POD_NAME` env var
  (`k8s/api-deployment.yaml`, set via the downward API) — this is what
  `k8s/watch-lb.sh` reads to prove requests are actually landing on
  different pods, not just show a static "it's probably load balanced".
- If port 80 (or 443) is already in use on the host/Codespace when
  `bootstrap.sh` runs `kind create cluster`, cluster creation fails — free
  the port or edit the `hostPort` values in `k8s/kind-config.yaml`.
- `k8s/hpa.yaml` shortens the scale-down stabilization window from
  Kubernetes' default (300s) to 60s purely so the demo is watchable without
  editing the recording. Don't carry that override into anything real.
- `pulse-api`'s CPU requests/limits (`k8s/api-deployment.yaml`) are set low
  (50m request) so a handful of load-generator replicas is enough to trip
  the 50% utilization target — this is a demo tuning, not a production
  sizing recommendation.
- `pulse-api`'s init container blocks until mongo is actually reachable, not
  just `Running`. On a cold cluster, mongo's entrypoint boots a temporary
  loopback-only instance to create the root user before restarting bound to
  `0.0.0.0` — combined with the first-time image pull, this can take several
  minutes. `bootstrap.sh` waits up to 450s for the `pulse-api` rollout to
  cover that; it's a one-time cost per cluster (fast on re-runs since images
  and mongo's init state are already cached).
