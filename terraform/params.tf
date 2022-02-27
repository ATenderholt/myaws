resource "aws_ssm_parameter" "endpoints" {
  for_each = var.endpoints

  name  = "/myaws/config/${each.key}"
  type  = "String"
  value = each.value
}
