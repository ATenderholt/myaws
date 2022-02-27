variable "endpoints" {
  type = object({
    iam    = string
    lambda = string
    s3     = string
    sqs    = string
    ssm    = string
    sts    = string
  })
}

variable "s3_use_path_style" {
  type = bool
}
