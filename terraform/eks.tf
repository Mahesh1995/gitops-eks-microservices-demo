# EKS cluster with one managed node group, using the community EKS module.
# EKS-managed add-ons (CoreDNS, kube-proxy, VPC CNI) are enabled for a working cluster.
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.20"

  cluster_name    = var.cluster_name
  cluster_version = var.cluster_version

  # Expose the API endpoint publicly for learning. Lock this down (or use a bastion)
  # for anything real — see docs/architecture.md security notes.
  cluster_endpoint_public_access = true

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  # Give the identity running `terraform apply` cluster-admin so you can kubectl in.
  enable_cluster_creator_admin_permissions = true

  cluster_addons = {
    coredns    = {}
    kube-proxy = {}
    vpc-cni    = {}
  }

  eks_managed_node_groups = {
    default = {
      instance_types = var.node_instance_types
      desired_size   = var.node_desired_size
      min_size       = var.node_min_size
      max_size       = var.node_max_size
      capacity_type  = "ON_DEMAND"
    }
  }
}
