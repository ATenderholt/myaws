resource "null_resource" "requests" {
  provisioner "local-exec" {
    command = "pip3.8 install -t ${path.root}/dependencies/requests/python requests"
  }
}

data "archive_file" "requests" {
  output_path = "packages/requests.zip"
  type        = "zip"
  source_dir  = "dependencies/requests"
  depends_on  = [null_resource.requests]
}

resource "aws_lambda_layer_version" "requests" {
  layer_name          = "myaws-requests"
  filename            = data.archive_file.requests.output_path
  compatible_runtimes = ["python3.8"]
  source_code_hash    = data.archive_file.requests.output_base64sha256
}
