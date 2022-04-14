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

data "aws_iam_policy_document" "fetch" {
  statement {
    sid     = "SQS"

    actions = [
      "sqs:DeleteMessage",
      "sqs:GetQueueAttributes",
      "sqs:ReceiveMessage"
    ]

    resources = [aws_sqs_queue.fetch.arn]
  }
}

resource "aws_iam_policy" "fetch" {
  name   = "myaws-fetch"
  policy = data.aws_iam_policy_document.fetch.json
}

resource "aws_iam_role_policy_attachment" "fetch" {
  role       = aws_iam_role.fetch.name
  policy_arn = aws_iam_policy.fetch.arn
}

resource "aws_iam_role_policy_attachment" "fetch_cloudwatch" {
  role       = aws_iam_role.fetch.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_lambda_function" "fetch" {
  function_name    = "myaws-fetch"
  role             = aws_iam_role.fetch.arn
  runtime          = "python3.8"
  handler          = "main.handler"
  filename         = data.archive_file.fetch.output_path
  layers           = [aws_lambda_layer_version.requests.arn]
  source_code_hash = data.archive_file.fetch.output_base64sha256
}

resource "aws_lambda_permission" "fetch" {
  statement_id  = "AllowFromSqs"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.fetch.function_name
  principal     = "sqs.amazonaws.com"
  source_arn    = aws_sqs_queue.fetch.arn
}

resource "aws_lambda_event_source_mapping" "fetch" {
  event_source_arn = aws_sqs_queue.fetch.arn
  function_name = aws_lambda_function.fetch.function_name
  batch_size = 1
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
  handler          = "main.handler"
  filename         = data.archive_file.copy_file.output_path
  source_code_hash = data.archive_file.copy_file.output_base64sha256

  environment {
    variables = {
      ENDPOINT = "http://localhost:8080"
    }
  }
}

resource "aws_lambda_permission" "copy_file" {
  statement_id  = "InvokeFromS3"
  action        = "lambda:InvokeFunction"
  function_name = "myaws-copy-file"
  principal     = "s3.amazonaws.com"
  source_arn    = aws_s3_bucket.main.arn
}

resource "aws_iam_role_policy_attachment" "copy_file" {
  role       = aws_iam_role.copy_file.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}
