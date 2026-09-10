# GitOps-Driven Microservices Demo on AWS EKS

A **learning-oriented, production-shaped** demo of a GitOps microservices platform on
AWS EKS. It mirrors the architecture and tooling of a full production-grade setup
(Terraform → EKS → ArgoCD → Helm/Kustomize → GitHub Actions → observability) but at a
scale you can read in an afternoon and grow one module at a time.

> Inspired by the "Online Boutique" style GitOps demos, deliberately scaled down to
> **4 small polyglot services** so the *platform* concerns (IaC, GitOps, CI/CD,
> networking, observability) stay front and centre — not the app code.

---

## What's inside

| Layer            | Tool                                   | Where                    |
|------------------|----------------------------------------|--------------------------|
| Infrastructure   | Terraform (VPC + EKS)                   | `terraform/`             |
| Microservices    | Go, Python/Flask, Node.js/Express      | `src/`                   |
| Packaging        | Kustomize (base + overlays), Helm      | `kustomize/`, `helm-chart/` |
| GitOps           | ArgoCD (App-of-nothing, points at Git) | `argocd/`                |
| CI/CD            | GitHub Actions (build → scan → push)   | `.github/workflows/`     |
| Ingress (L7)     | Gateway API → AWS ALB                   | `gateway-api/`           |
| DNS              | External DNS → Route53                  | `external-dns/`          |
| Observability    | kube-prometheus-stack (Helm values)    | `observability/`         |

### The application

A trivial shopping demo. The **frontend** aggregates the backends, and **checkout**
orchestrates cart + product-catalog to turn a cart into a priced order:

```
                +-------------------+
   browser ---> |  frontend (Go)    |
                +---------+---------+
                          |  HTTP (REST)
             +------------+-------------+
             |                          |
   +---------v---------+     +----------v----------+
   | product-catalog   |     | cart (Node.js)      |
   | (Python / Flask)  |     | + Redis (optional)  |
   +---------^---------+     +----------^----------+
             |                          |
             +-----------+--------------+
                         |
                +--------+---------+
                | checkout (Go)    |  POST /checkout/{user} -> priced order
                +------------------+
```

REST (not gRPC) is used on purpose — it keeps the services readable. The production
reference uses gRPC; swapping the transport is a great later exercise.

---

## Agile learning roadmap

Work through it as small, independently testable modules. Each step is self-contained
and leaves you with something that runs.

- **Module 0 — Run it locally (no cloud).** `docker compose up` (see `docs/runbook.md`)
  or run each service with its language runtime. Confirm the frontend renders products
  and can add to cart.
- **Module 1 — Containers & CI.** Understand each `Dockerfile`. Push to GitHub and watch
  `.github/workflows/ci.yaml` build, Trivy-scan, and publish images to GHCR.
- **Module 2 — Kubernetes by hand.** `kubectl apply -k kustomize/overlays/dev` against a
  local cluster (kind/minikube) *or* EKS. Learn Services, Deployments, probes, HPA.
- **Module 3 — Infrastructure as Code.** `terraform apply` in `terraform/` to stand up a
  VPC + EKS cluster. Read every resource; this is the AWS-cert-relevant part.
- **Module 4 — GitOps with ArgoCD.** Install ArgoCD, apply `argocd/application.yaml`, and
  let Git become the single source of truth. Change a replica count in Git → watch it sync.
- **Module 5 — Observability.** Install kube-prometheus-stack with
  `observability/kube-prometheus-stack.values.yaml`. Explore Grafana dashboards & alerts.
- **Module 6 — Expose it (Gateway API + DNS).** Install the AWS Load Balancer Controller
  and Gateway API CRDs (IRSA from `terraform/irsa.tf`), apply `gateway-api/` to provision an
  ALB, then deploy `external-dns/` so a real hostname in Route53 points at it automatically.
- **Module 7 — Grow it further.** Switch REST→gRPC, add Redis persistence, add Argo Image
  Updater, try a canary with Argo Rollouts, move Terraform state to S3. See
  `docs/architecture.md` for stretch goals.

---

## Quick start (local)

```bash
# 1. Run the whole app locally with Docker Compose
docker compose up --build
# frontend on http://localhost:8080

# 2. Or deploy to a local Kubernetes cluster
kind create cluster --name gitops-demo
kubectl apply -k kustomize/overlays/dev
kubectl -n gitops-demo port-forward svc/frontend 8080:80
```

Full step-by-step (including the AWS EKS path) is in [`docs/runbook.md`](docs/runbook.md).

---

## Repository layout

```
.
├── src/                     # microservice source + Dockerfiles
│   ├── frontend/            # Go — web UI + API gateway
│   ├── product-catalog/     # Python/Flask — product data
│   ├── cart/                # Node.js/Express — cart (+ Redis)
│   └── checkout/            # Go — order orchestration
├── kustomize/               # GitOps source of truth
│   ├── base/                # shared manifests
│   └── overlays/{dev,prod}/ # per-environment patches
├── helm-chart/              # alternative packaging (learn Helm)
├── argocd/                  # ArgoCD Project + Application
├── terraform/               # AWS VPC + EKS + IRSA roles
├── gateway-api/             # Gateway API → AWS ALB (L7 ingress)
├── external-dns/            # External DNS → Route53
├── observability/           # Prometheus/Grafana stack values
├── .github/workflows/       # CI: build → Trivy scan → push to GHCR
├── docs/                    # architecture + runbook
├── docker-compose.yaml      # Module 0 local run
└── Makefile                 # common commands
```

## Prerequisites

`docker`, `kubectl`, `helm`, `kustomize`, `terraform`, `aws` CLI, and (optional)
`kind`/`minikube` for local clusters. Versions are pinned where it matters
(`terraform/versions.tf`).

## License

MIT — this is a teaching scaffold; use it however helps you learn.
