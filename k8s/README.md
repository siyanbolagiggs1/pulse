# Local HPA demo (kind, inside Codespaces)

Self-contained kind cluster running `pulse-api` + mongo + redis, with a
HorizontalPodAutoscaler watching `pulse-api`'s CPU. No local Docker needed —
this runs entirely inside the Codespace via `docker-in-docker`.

## Setup

```bash
bash k8s/bootstrap.sh
```

Takes ~5-10 minutes on first run (image build + metrics-server install).
Re-running is fast — it reuses the existing cluster.

## Demo script

Open two terminals.

Terminal 1 — leave this running, it's the money shot:

```bash
kubectl get hpa pulse-api -w
```

Terminal 2 — drive the traffic:

```bash
# ramp up traffic
kubectl scale deployment/load-generator --replicas=6

# watch terminal 1: TARGETS climbs past 50%, REPLICAS goes 1 -> 2/3
# (metrics-server scrapes every 15s, HPA re-evaluates every 15s — expect
# a scale-up within ~30-60s of ramping up)

# stop traffic
kubectl scale deployment/load-generator --replicas=0

# watch terminal 1: REPLICAS drops back to 1 after the stabilization
# window (60s here — see k8s/hpa.yaml, default is 300s in real clusters)
```

`kubectl get pods -w` in a third terminal is a nice visual too — shows the
new `pulse-api` pod actually being created/terminated, not just the HPA's
replica count.

## Cleanup

```bash
kind delete cluster --name pulse
```

## Notes

- `k8s/hpa.yaml` shortens the scale-down stabilization window from
  Kubernetes' default (300s) to 60s purely so the demo is watchable without
  editing the recording. Don't carry that override into anything real.
- `pulse-api`'s CPU requests/limits (`k8s/api-deployment.yaml`) are set low
  (50m request) so a handful of load-generator replicas is enough to trip
  the 50% utilization target — this is a demo tuning, not a production
  sizing recommendation.
