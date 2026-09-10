# Gateway API (AWS ALB)

These manifests expose the app through the **Kubernetes Gateway API**, implemented by the
**AWS Load Balancer Controller**, which provisions an **Application Load Balancer** (ALB).

```
Internet ──> ALB (from Gateway) ──> HTTPRoute ──┬─ /api/checkout ─> checkout:4000
                                                └─ /            ─> frontend:80
```

| File                 | Kind          | Scope      | Purpose                                  |
|----------------------|---------------|------------|------------------------------------------|
| `gatewayclass.yaml`  | GatewayClass  | cluster    | Selects the AWS ALB controller           |
| `gateway.yaml`       | Gateway       | namespaced | Provisions the ALB + HTTP listener       |
| `httproute.yaml`     | HTTPRoute     | namespaced | Path routing to `frontend` / `checkout`  |

## Prerequisites

1. **Gateway API CRDs** installed in the cluster:
   ```bash
   kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/standard-install.yaml
   ```
2. **AWS Load Balancer Controller v2.13+** with Gateway API support enabled, using the
   IRSA role from `terraform/irsa.tf`. See `../docs/runbook.md` (Module 7).

## Apply

```bash
kubectl apply -f gateway-api/gatewayclass.yaml
kubectl apply -f gateway-api/gateway.yaml
kubectl apply -f gateway-api/httproute.yaml

# Find the ALB hostname the controller created:
kubectl -n gitops-demo get gateway shop-gateway -o jsonpath='{.status.addresses[0].value}'
```

Set the real domain in `httproute.yaml` (`hostnames:`) before applying, so External DNS
can create the Route53 record for it.
