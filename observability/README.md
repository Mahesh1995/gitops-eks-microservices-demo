# Observability

The stack is the community **kube-prometheus-stack** (Prometheus + Grafana +
Alertmanager + node/pod exporters), configured by
[`kube-prometheus-stack.values.yaml`](kube-prometheus-stack.values.yaml).

## Install

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install kube-prom-stack prometheus-community/kube-prometheus-stack \
  -n monitoring --create-namespace \
  -f observability/kube-prometheus-stack.values.yaml
```

## Access Grafana

```bash
kubectl -n monitoring port-forward svc/kube-prom-stack-grafana 3001:80
# http://localhost:3001   user: admin   pass: prom-operator (from the values file)
```

## What gets scraped

Each of our services exposes `/metrics` in Prometheus text format and carries
`prometheus.io/scrape` pod annotations. The `annotated-pods` scrape config in the values
file discovers them, so you'll see:

- `frontend_requests_total`, `frontend_request_errors_total`
- `product_catalog_requests_total`, `product_catalog_products`
- `cart_requests_total`, `cart_items_added_total`

Explore them in Grafana → Explore, or build a dashboard. Node/pod/cluster dashboards ship
with the chart automatically.

## Next steps

- Add Alertmanager → Slack routing (commented in the values file).
- Add a `ServiceMonitor` per service instead of annotation scraping (the more idiomatic
  Prometheus-Operator way).
- Add centralized logging (Elasticsearch + Filebeat, or Loki) as in the reference project.
