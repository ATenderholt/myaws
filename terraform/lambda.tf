data "archive_file" "fetch" {
  output_path = "packages/fetch.zip"
  type        = "zip"
  source_dir  = "lambdas/fetch"
}

data "aws_iam_policy_document" "lambda" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      identifiers = ["lambda.amazonaws.com"]
      type        = "Service"
    }
  }
}

resource "aws_iam_role" "fetch" {
  name = "myaws-fetch"
  assume_role_policy = data.aws_iam_policy_document.lambda.json
}

resource "aws_lambda_function" "fetch" {
  function_name    = "myaws-fetch"
  role             = aws_iam_role.fetch.arn
  runtime          = "python3.8"
  handler          = "main.handle"
  filename         = data.archive_file.fetch.output_path
  layers           = [aws_lambda_layer_version.requests.arn]
  source_code_hash = data.archive_file.fetch.output_base64sha256
}

data "archive_file" "copy_file" {
  output_path = "packages/copyFile.zip"
  type        = "zip"
  source_dir  = "lambdas/copyFile"
}

resource "aws_iam_role" "copy_file" {
  name = "myaws-copy-file"
  assume_role_policy = data.aws_iam_policy_document.lambda.json
}

resource "aws_lambda_function" "copy_file" {
  function_name    = "myaws-copy-file"
  role             = aws_iam_role.copy_file.arn
  runtime          = "python3.8"
  handler          = "main.handle"
  filename         = data.archive_file.copy_file.output_path
  source_code_hash = data.archive_file.copy_file.output_base64sha256
}
