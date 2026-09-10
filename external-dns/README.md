# External DNS

External DNS automatically creates and updates **Route53** records so a friendly hostname
(e.g. `shop.example.com`) always points at the ALB that the Gateway provisioned — no manual
DNS work.

```
HTTPRoute (hostnames: shop.example.com)
        │  watched by
        ▼
   External DNS ──> Route53 (A/ALIAS record ─> ALB)
```

## How it works here

- `--source=gateway-httproute` — it reads hostnames from Gateway API HTTPRoutes.
- `--provider=aws` with an **IRSA** role (`terraform/irsa.tf` → `external_dns_role_arn`)
  that grants exactly the Route53 permissions it needs.
- `--policy=upsert-only` — it will create/update but never delete records, which is the
  safer default while learning.
- `--txt-owner-id` — External DNS writes a companion TXT record so it only ever manages
  records it created.

## Apply

```bash
# 1. Put the IRSA role ARN on the ServiceAccount, and set your zone:
#    - eks.amazonaws.com/role-arn annotation
#    - --domain-filter=<your-zone>   (and --txt-owner-id)
kubectl apply -f external-dns/external-dns.yaml
kubectl -n external-dns logs deploy/external-dns -f   # watch it reconcile records
```

## Verify

```bash
# After the HTTPRoute is applied, a record should appear:
aws route53 list-resource-record-sets --hosted-zone-id <ZONE_ID> \
  --query "ResourceRecordSets[?Name=='shop.example.com.']"
```

> Prerequisite: a **public Route53 hosted zone** for your domain must already exist, and
> your domain's registrar must delegate to that zone's name servers.
