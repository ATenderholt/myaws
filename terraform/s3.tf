resource "aws_s3_bucket" "main" {
  bucket = "myaws-files"
  tags = {
    Cost = "myaws-files"
  }
}
