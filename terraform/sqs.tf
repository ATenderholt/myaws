resource "aws_sqs_queue" "fetch" {
  name = "myaws-fetch"
  tags = {
    foo = "bar"
  }
}
