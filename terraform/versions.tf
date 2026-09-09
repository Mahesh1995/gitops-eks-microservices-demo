terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.60"
    }
  }

  # When you're ready for team/remote state, uncomment and create the bucket +
  # DynamoDB lock table first (Module 3 → S3 backend stretch goal):
  #
  # backend "s3" {
  #   bucket         = "your-tfstate-bucket"
  #   key            = "gitops-eks-demo/terraform.tfstate"
  #   region         = "ap-south-1"
  #   dynamodb_table = "your-tfstate-lock"
  #   encrypt        = true
  # }
}
