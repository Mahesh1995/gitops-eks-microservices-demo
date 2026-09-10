# Runbook

Step-by-step for each module. Commands assume you're at the repo root.

## Module 0 — Run locally (no cloud)

### Option A: Docker Compose (easiest)

```bash
docker compose up --build
# open http://localhost:8080
```

### Option B: run each service natively

```bash
# product-catalog (Python)
cd src/product-catalog && pip install -r requirements.txt && python app.py   # :5000

# cart (Node.js) — in a second terminal
cd src/cart && npm install && npm start                                       # :3000

# frontend (Go) — in a third terminal
cd src/frontend && \
  PRODUCT_CATALOG_ADDR=localhost:5000 CART_ADDR=localhost:3000 go run .        # :8080
```

Smoke test:

```bash
curl -s localhost:5000/products | jq .
curl -s -XPOST localhost:3000/cart/alice/items -d '{"product_id":"OLJCESPC7Z","qty":2}' -H 'content-type: application/json'
curl -s localhost:3000/cart/alice | jq .
curl -s localhost:8080/healthz
# checkout turns alice's cart into a priced order:
curl -s -XPOST localhost:4000/checkout/alice | jq .
```

> To run `checkout` natively (Option B), start it with:
> `cd src/checkout && CART_ADDR=localhost:3000 PRODUCT_CATALOG_ADDR=localhost:5000 go run .`  (:4000)

## Module 1 — CI to GHCR

1. Push the repo to GitHub (already done if you're reading this there).
2. Open the **Actions** tab; `ci.yaml` runs on every push.
3. It detects changed services under `src/`, builds each image, runs a **Trivy** scan,
   and pushes to `ghcr.io/<owner>/<service>:sha-<commit>`.
4. Make GHCR packages public (or configure imagePullSecrets) before the cluster pulls them.

> The workflow needs no extra secrets — it uses the built-in `GITHUB_TOKEN` with
> `packages: write`.

## Module 2 — Kubernetes (local cluster)

```bash
kind create cluster --name gitops-demo          # or: minikube start
kubectl apply -k kustomize/overlays/dev
kubectl -n gitops-demo get pods,svc
kubectl -n gitops-demo port-forward svc/frontend 8080:80
# open http://localhost:8080
```

Tear down: `kind delete cluster --name gitops-demo`.

## Module 3 — Terraform: VPC + EKS

```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars     # edit region, cluster_name, etc.
terraform init
terraform plan
terraform apply                                   # ~15 min; creates real AWS resources ($$)
aws eks update-kubeconfig --name "$(terraform output -raw cluster_name)" --region "$(terraform output -raw region)"
kubectl get nodes
```

> Costs money. Run `terraform destroy` when done. The default node group is small
> (2× t3.medium) to keep the bill low.

## Module 4 — ArgoCD (GitOps)

```bash
# Install ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
kubectl -n argocd rollout status deploy/argocd-server

# Point ArgoCD at THIS repo (edit repoURL in argocd/application.yaml first!)
kubectl apply -f argocd/project.yaml
kubectl apply -f argocd/application.yaml

# Watch it sync
kubectl -n argocd get applications
```

GitOps test: change `replicas` in `kustomize/overlays/dev`, commit, push — ArgoCD
reconciles the cluster to match within a minute (or click **Sync**).

Access the UI:

```bash
kubectl -n argocd port-forward svc/argocd-server 8081:443
# user: admin  — password:
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d
```

## Module 5 — Observability

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install kube-prom-stack prometheus-community/kube-prometheus-stack \
  -n monitoring --create-namespace \
  -f observability/kube-prometheus-stack.values.yaml

kubectl -n monitoring port-forward svc/kube-prom-stack-grafana 3001:80
# Grafana: http://localhost:3001  (admin / prom-operator by default)
```

## Module 6 — Expose it (Gateway API + External DNS)

> AWS-only; requires the EKS cluster from Module 3 and the IRSA roles from
> `terraform/irsa.tf` (run `terraform apply` again if you added them after Module 3).

```bash
# Role ARNs created by Terraform:
cd terraform
LBC_ROLE=$(terraform output -raw aws_lb_controller_role_arn)
EDNS_ROLE=$(terraform output -raw external_dns_role_arn)
cd ..

# 1. Gateway API CRDs
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/standard-install.yaml

# 2. AWS Load Balancer Controller (v2.13+ for Gateway API support), via Helm + IRSA
helm repo add eks https://aws.github.io/eks-charts && helm repo update
helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
  -n kube-system \
  --set clusterName="$(cd terraform && terraform output -raw cluster_name)" \
  --set serviceAccount.create=true \
  --set serviceAccount.name=aws-load-balancer-controller \
  --set "serviceAccount.annotations.eks\.amazonaws\.com/role-arn=${LBC_ROLE}"

# 3. The Gateway + route (edit the hostname in gateway-api/httproute.yaml first)
kubectl apply -f gateway-api/gatewayclass.yaml
kubectl apply -f gateway-api/gateway.yaml
kubectl apply -f gateway-api/httproute.yaml
kubectl -n gitops-demo get gateway shop-gateway -o wide   # wait for the ALB address

# 4. External DNS (put $EDNS_ROLE on the SA + set --domain-filter in the manifest)
kubectl apply -f external-dns/external-dns.yaml
kubectl -n external-dns logs deploy/external-dns -f       # watch it create the Route53 record
```

Then browse to your hostname (e.g. `http://shop.example.com`). Tear down the ALB by
deleting the Gateway (`kubectl delete -f gateway-api/gateway.yaml`) before
`terraform destroy`, so no load balancer is orphaned.

## Common issues

- **Pods `ImagePullBackOff`** → GHCR package is private; make it public or add an
  imagePullSecret to the `gitops-demo` namespace.
- **ArgoCD `OutOfSync` forever** → check `repoURL`/`targetRevision`/`path` in
  `argocd/application.yaml` match your fork.
- **`terraform apply` auth error** → `aws sts get-caller-identity` first; your CLI creds
  must be valid and have EKS/VPC/IAM permissions.
