variable "endpoints" {
  type = object({
    lambda = string
    s3     = string
    sqs    = string
  })
}
