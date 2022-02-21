provider "aws" {
  # ... potentially other provider configuration ...

  region = "us-west-2"

  endpoints {
    lambda = var.endpoints.lambda
    s3     = var.endpoints.s3
    sqs    = var.endpoints.sqs
  }
}
