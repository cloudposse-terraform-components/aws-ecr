locals {
  enabled = module.this.enabled
}

module "full_access" {
  source = "../account-map/modules/roles-to-principals"

  role_map = var.read_write_account_role_map

  context = module.this.context
}

module "readonly_access" {
  source = "../account-map/modules/roles-to-principals"

  role_map = var.read_only_account_role_map

  context = module.this.context
}

locals {
  ecr_user_arn = join("", aws_iam_user.ecr[*].arn)
}

module "ecr" {

  source  = "cloudposse/ecr/aws"
  version = "1.0.0"

  repository_name = var.repository_name

  principals_full_access     = compact(concat(module.full_access.principals, [local.ecr_user_arn]))
  principals_readonly_access = module.readonly_access.principals
  principals_lambda          = var.principals_lambda
  scan_images_on_push        = var.scan_images_on_push
  force_delete               = var.force_delete

  replication_configurations = var.replication_configurations
  lifecycle_rules            = var.lifecycle_rules

  context = module.this.context
}

data "aws_secretsmanager_secret" "cache_credentials" {
  for_each = local.enabled ? {
    for key, rule in var.pull_through_cache_rules :
    key => rule.secret
    if length(rule.secret) > 0
  } : {}

  name = each.value
}

resource "aws_ecr_pull_through_cache_rule" "this" {
  for_each = local.enabled ? var.pull_through_cache_rules : {}

  ecr_repository_prefix = each.key
  upstream_registry_url = each.value.registry
  credential_arn        = length(each.value.secret) > 0 ? data.aws_secretsmanager_secret.cache_credentials[each.key].arn : null
}
