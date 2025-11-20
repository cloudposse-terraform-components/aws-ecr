locals {
  enabled = module.this.enabled
}

module "full_access" {
  source = "github.com/cloudposse-terraform-components/aws-account-map//src/modules/roles-to-principals?ref=v1.536.1"

  role_map = var.read_write_account_role_map

  tenant      = var.account_map_enabled ? module.iam_roles.global_tenant_name : null
  environment = var.account_map_enabled ? module.iam_roles.global_environment_name : null
  stage       = var.account_map_enabled ? module.iam_roles.global_stage_name : null

  account_map_bypass   = !var.account_map_enabled
  account_map_defaults = var.account_map

  context = module.this.context
}

module "readonly_access" {
  source = "github.com/cloudposse-terraform-components/aws-account-map//src/modules/roles-to-principals?ref=v1.536.1"

  role_map = var.read_only_account_role_map

  tenant      = var.account_map_enabled ? module.iam_roles.global_tenant_name : null
  environment = var.account_map_enabled ? module.iam_roles.global_environment_name : null
  stage       = var.account_map_enabled ? module.iam_roles.global_stage_name : null

  account_map_bypass   = !var.account_map_enabled
  account_map_defaults = var.account_map

  context = module.this.context
}

locals {
  ecr_user_arn = join("", aws_iam_user.ecr[*].arn)
}

module "ecr" {
  source  = "cloudposse/ecr/aws"
  version = "1.0.0"

  protected_tags                        = var.protected_tags
  protected_tags_keep_count             = var.protected_tags_keep_count
  enable_lifecycle_policy               = var.enable_lifecycle_policy
  default_lifecycle_rules_settings      = var.default_lifecycle_rules_settings
  image_names                           = var.images
  image_tag_mutability                  = var.image_tag_mutability
  image_tag_mutability_exclusion_filter = var.image_tag_mutability_exclusion_filter
  max_image_count                       = var.max_image_count
  principals_full_access                = compact(concat(module.full_access.principals, [local.ecr_user_arn]))
  principals_readonly_access            = module.readonly_access.principals
  principals_lambda                     = var.principals_lambda
  scan_images_on_push                   = var.scan_images_on_push
  use_fullname                          = false
  replication_configurations            = var.replication_configurations

  custom_lifecycle_rules = var.custom_lifecycle_rules

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

data "aws_caller_identity" "current" {
  count = local.enabled ? 1 : 0
}

resource "aws_ecr_registry_policy" "this" {
  for_each = toset(local.enabled && length(var.pull_through_cache_rules) > 0 ? ["true"] : [])
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ecr:BatchGetImage",
          "ecr:GetDownloadUrlForLayer",
          "ecr:GetImageCopyStatus",
          "ecr:BatchImportUpstreamImage"
        ]
        Principal = {
          AWS = distinct(compact(concat(module.full_access.principals, module.readonly_access.principals, [local.ecr_user_arn])))
        }
        Resource = format("arn:aws:ecr:%s:%s:repository/*", var.region, one(data.aws_caller_identity.current.*.account_id))
      }
    ]
  })
}
