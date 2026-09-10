# IRSA (IAM Roles for Service Accounts) for the two controllers that need AWS API access:
#   - AWS Load Balancer Controller  (provisions ALBs for the Gateway API)
#   - External DNS                  (manages Route53 records)
#
# IRSA maps a Kubernetes ServiceAccount to an IAM role via the cluster's OIDC provider, so
# pods get scoped AWS permissions without any static credentials. The community module
# ships ready-made policies for both controllers.

# ---- AWS Load Balancer Controller ---------------------------------------
module "lb_controller_irsa" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks"
  version = "~> 5.44"

  role_name                              = "${var.cluster_name}-aws-lb-controller"
  attach_load_balancer_controller_policy = true

  oidc_providers = {
    main = {
      provider_arn               = module.eks.oidc_provider_arn
      namespace_service_accounts = ["kube-system:aws-load-balancer-controller"]
    }
  }
}

# ---- External DNS -------------------------------------------------------
module "external_dns_irsa" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks"
  version = "~> 5.44"

  role_name                  = "${var.cluster_name}-external-dns"
  attach_external_dns_policy = true

  oidc_providers = {
    main = {
      provider_arn               = module.eks.oidc_provider_arn
      namespace_service_accounts = ["external-dns:external-dns"]
    }
  }
}

# ---- Outputs: paste these ARNs into the ServiceAccount annotations -------
output "aws_lb_controller_role_arn" {
  description = "IRSA role ARN for the AWS Load Balancer Controller ServiceAccount."
  value       = module.lb_controller_irsa.iam_role_arn
}

output "external_dns_role_arn" {
  description = "IRSA role ARN for the external-dns ServiceAccount."
  value       = module.external_dns_irsa.iam_role_arn
}
