provider "aws" {
  region = var.region

  default_tags {
    tags = {
      Project     = "gitops-eks-microservices-demo"
      ManagedBy   = "terraform"
      Environment = var.environment
    }
  }
}
