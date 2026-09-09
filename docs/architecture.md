# Architecture

## Overview

This demo is intentionally split into two concerns:

1. **The application** — three tiny microservices that together form a shopping demo.
2. **The platform** — everything that provisions, deploys, exposes, and observes them.

The learning value is almost entirely in #2. The app is kept small so that the platform
mechanics (IaC, GitOps, CI/CD, networking, observability) are never obscured by business
logic.

## Application services

| Service           | Language        | Responsibility                                  | Port |
|-------------------|-----------------|-------------------------------------------------|------|
| `frontend`        | Go (net/http)   | Serves the UI, aggregates the two backends      | 8080 |
| `product-catalog` | Python / Flask  | Returns the product list from a JSON file       | 5000 |
| `cart`            | Node.js/Express | Add/list cart items, optional Redis persistence | 3000 |

All communication is REST/JSON over HTTP. Each service exposes:

- `GET /healthz` — liveness (process is up)
- `GET /readyz`  — readiness (dependencies reachable)
- `GET /metrics` — Prometheus metrics (frontend & cart expose basic counters)

### Why REST instead of gRPC?

The production reference uses gRPC + protobufs across 11 services. That is excellent but
adds a code-generation step and binary transport that hide the request flow from a
beginner. Once Modules 0–5 are comfortable, swapping `product-catalog` to gRPC is a
focused, high-value exercise (Module 6).

## Data flow

```
Browser
  │  GET /
  ▼
frontend (Go)
  ├─ GET  product-catalog:5000/products   → list of products
  └─ POST cart:3000/cart/{user}/items      → add to cart
        cart ──(optional)── Redis          → persistence
```

Only `cart` is stateful, and even then Redis is optional (it falls back to in-memory).
This mirrors the reference's deliberate choice: keep services stateless so the platform,
not the data layer, is what you study.

## Platform architecture (AWS EKS path)

```
        GitHub (source of truth)
          │              │
   git push │            │ ArgoCD watches
          ▼              ▼
 GitHub Actions      ArgoCD (in-cluster)
  build+scan+push        │ sync
          │              ▼
        GHCR ────────► EKS cluster ──► kube-prometheus-stack
     (container         (Deployments,      (Prometheus, Grafana,
      registry)          Services, HPA)      Alertmanager)
                             ▲
                             │ provisioned by
                        Terraform (VPC + EKS)
```

- **Terraform** provisions the network (VPC, subnets, NAT) and the EKS control plane +
  managed node group. State is local by default; an S3 backend is shown commented in
  `terraform/versions.tf` for when you're ready.
- **GitHub Actions** builds only the services whose code changed, scans images with Trivy,
  and pushes to GHCR tagged `sha-<commit>`.
- **ArgoCD** watches this repo's `kustomize/overlays/<env>` path and reconciles the
  cluster to match Git. No `kubectl apply` in day-to-day operation.
- **kube-prometheus-stack** provides Prometheus, Grafana and Alertmanager out of the box.

## Environments

`kustomize/base` holds the shared manifests. Overlays patch them:

- `overlays/dev` — 1 replica, low resource requests, `latest`-ish tags, debug logging.
- `overlays/prod` — 2+ replicas, HPA enabled, tighter resources, info logging.

ArgoCD runs one `Application` per environment (only `dev` is wired by default).

## Stretch goals (Module 6)

- Switch `frontend ↔ product-catalog` to **gRPC** with a shared `.proto`.
- Add **Redis** as a real dependency with a StatefulSet + PVC.
- Add **AWS Load Balancer Controller + Gateway API** for L7 ingress.
- Add **External DNS** to manage Route53 records from `HTTPRoute` annotations.
- Add **Argo Image Updater** so new GHCR tags auto-bump the overlay.
- Add a **canary** rollout with Argo Rollouts.
- Move Terraform state to an **S3 backend + DynamoDB lock**.

## Security notes (learning, not hardening)

This scaffold favours clarity over hardening. Before anything real: pin image digests,
add NetworkPolicies, drop root in containers (already done via `USER` where possible),
enable IRSA for pod IAM, and never commit `*.tfvars` or kubeconfigs (see `.gitignore`).
