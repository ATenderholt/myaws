resource "aws_s3_bucket" "main" {
  bucket = "myaws-files"

  // TODO : tagging
  // --- Request PUT "/myaws-files" ---
  // <Tagging xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><TagSet><Tag><Key>Cost</Key><Value>myaws-files</Value></Tag></TagSet></Tagging>
#  tags = {
#    Cost = "myaws-files"
#  }
}
