# k8s-topology-dashboard

A read-only **Kubernetes topology dashboard** — a graph-first view (Headlamp × Visio hybrid) that renders your cluster objects as a navigable graph and lets you drill down into live details.

```
Namespace  ──  Deployment  ──  ReplicaSet  ──  PodSet
                                                  │
Service (selects) ────────────────────────────────┘
Ingress  ──  Service
```

## Architecture

| Service | Path | Port |
|---------|------|------|
| Go REST API | `api/` | 8080 |
| Vite + React + Cytoscape.js UI | `ui/` | 5173 (dev) / 80 (container) |

---

## Local development

### Prerequisites

- Go ≥ 1.21
- Node.js ≥ 18
- A kubeconfig pointing at a live cluster (or `KUBECONFIG` env var)

### Run

```bash
# 1. Start the API
make api-run          # uses $KUBECONFIG or ~/.kube/config

# 2. Start the UI (in a separate terminal)
make ui-dev           # opens http://localhost:5173
```

You can also start them manually:

```bash
# API
cd api && go run ./cmd/dashboard-api --kubeconfig ~/.kube/config --cors-origin http://localhost:5173

# UI
cd ui && npm install && npm run dev
```

### Environment variables (API)

| Variable | Default | Description |
|----------|---------|-------------|
| `KUBECONFIG` | — | Path to kubeconfig (overridden by `--kubeconfig` flag) |
| `ADDR` | `:8080` | Listen address |
| `CORS_ORIGIN` | — | Allowed CORS origin (e.g. `http://localhost:5173`) |
| `MAX_NODES` | `500` | Truncation cap for nodes |
| `MAX_EDGES` | `2000` | Truncation cap for edges |

### Environment variables (UI)

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_BASE_URL` | `http://localhost:8080` | API base URL used by the browser |

---

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Health check |
| GET | `/api/graph` | Topology graph (nodes + edges) |
| GET | `/api/object/{kind}/{namespace}/{name}` | Object detail + sanitized YAML |

### `/api/graph` query parameters

| Param | Default | Description |
|-------|---------|-------------|
| `namespaces` | `*` | Comma-separated namespace list, or `*` for all |
| `q` | — | Substring filter on node name/namespace |
| `includePods` | `false` | Show real Pod nodes instead of collapsed PodSets |

---

## In-cluster deployment

```bash
# 1. Apply manifests (API RBAC + Deployment/Service + UI Deployment/Service)
kubectl apply -f deploy/

# 2. Wait for pods to be ready
kubectl rollout status deployment/k8s-topology-api
kubectl rollout status deployment/k8s-topology-ui

# 3. Access the UI
kubectl port-forward svc/k8s-topology-ui 8090:80
# open http://localhost:8090

# (Optional) Access the API directly
kubectl port-forward svc/k8s-topology-api 8080:8080
```

The API service is reachable from the UI pod at `http://k8s-topology-api:8080` within the cluster.  
Set `VITE_API_BASE_URL=http://k8s-topology-api:8080` in the UI deployment if you need the browser to call the API on a separate origin.

---

## Safety

- **Read-only** — the API only calls `get`, `list`, `watch` on allowed resource types.
- **No Secrets** — `kind=Secret` is rejected (HTTP 400). Secret data is never returned.
- **RBAC least-privilege** — the `ClusterRole` in `deploy/rbac.yaml` grants access only to:  
  `namespaces`, `nodes`, `pods`, `services`, `deployments`, `replicasets`, `ingresses`.

---

## Makefile targets

```
make api-run     Run API locally
make api-build   Build API binary
make ui-dev      Run UI dev server
make ui-build    Build UI for production
make build       Build both
```
