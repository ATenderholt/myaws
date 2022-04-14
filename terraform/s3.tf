resource "aws_s3_bucket" "main" {
  bucket = "myaws-files"

  // TODO : tagging
  // --- Request PUT "/myaws-files" ---
  // <Tagging xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><TagSet><Tag><Key>Cost</Key><Value>myaws-files</Value></Tag></TagSet></Tagging>
#  tags = {
#    Cost = "myaws-files"
#  }
}

resource "aws_s3_bucket_notification" "main" {
  bucket = aws_s3_bucket.main.id

  lambda_function {
#    lambda_function_arn = "arn:aws:lambda:us-west-2:271828182845:function:myaws-copy-file"
    lambda_function_arn = "arn:aws:lambda:us-west-2:675294739408:function:myaws-copy-file"
    events              = ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
    filter_prefix       = "AWSLogs/"
    filter_suffix       = ".log"
  }
}