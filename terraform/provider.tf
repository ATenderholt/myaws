provider "aws" {
  # ... potentially other provider configuration ...

  region            = "us-west-2"
  s3_use_path_style = var.s3_use_path_style

  endpoints {
    iam    = var.endpoints.iam
    lambda = var.endpoints.lambda
    s3     = var.endpoints.s3
    sqs    = var.endpoints.sqs
    ssm    = var.endpoints.ssm
#    sts    = var.endpoints.sts
  }
}

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
    }
  }
}