variable "region" {
  type        = string
  description = "AWS Region"
}

variable "repository_name" {
  type        = string
  description = "Name of the ECR repository to create"
}



variable "read_write_account_role_map" {
  type        = map(list(string))
  description = "Map of `account:[role, role...]` for write access. Use `*` for role to grant access to entire account"
}

variable "read_only_account_role_map" {
  type        = map(list(string))
  description = "Map of `account:[role, role...]` for read-only access. Use `*` for role to grant access to entire account"
  default     = {}
}

variable "ecr_user_enabled" {
  type        = bool
  description = "Enable/disable the provisioning of the ECR user (for CI/CD systems that don't support assuming IAM roles to access ECR, e.g. Codefresh)"
  default     = false
}

variable "scan_images_on_push" {
  type        = bool
  description = "Indicates whether images are scanned after being pushed to the repository"
  default     = false
}

variable "principals_lambda" {
  type        = list(string)
  description = "Principal account IDs of Lambdas allowed to consume ECR"
  default     = []
}

variable "force_delete" {
  type        = bool
  description = "Whether to delete the repository even if it contains images"
  default     = null
}

variable "pull_through_cache_rules" {
  type = map(object({
    registry = string
    secret   = optional(string, "")
  }))
  description = "Map of pull through cache rules to configure"
  default     = {}
}

variable "replication_configurations" {
  type = list(object({
    rules = list(object({
      destinations = list(object({
        region      = string
        registry_id = string
      }))
      repository_filters = list(object({
        filter      = string
        filter_type = string
      }))
    }))
  }))
  description = "Replication configuration for a registry. See [Replication Configuration](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecr_replication_configuration#replication-configuration)."
  default     = []
}

variable "lifecycle_rules" {
  description = "Custom lifecycle rules to override or complement the default ones"
  type = list(object({
    priority    = number
    description = optional(string)
    selection = list(object({
      tag_status       = string
      count_type       = string
      count_number     = number
      count_unit       = optional(string)
      tag_pattern_list = optional(list(string))
      tag_prefix_list  = optional(list(string))
    }))
    action = object({
      type = string
    })
  }))
  default = [
    {
      priority    = 10
      description = "Default lifecycle rule"
      selection = [{
        tag_status = "any"
        count_type = "imageCountMoreThan"
        # AWS limit https://docs.aws.amazon.com/AmazonECR/latest/userguide/service-quotas.html
        count_number = 20000
      }]
      action = {
        type = "expire"
      }
    }
  ]

  validation {
    condition = alltrue(flatten([
      for rule in var.lifecycle_rules :
      [for selection in rule.selection :
      contains(["tagged", "untagged", "any"], selection.tag_status)]
    ]))
    error_message = "Valid values for tag_status are: tagged, untagged, or any."
  }
  validation {
    condition = alltrue(flatten([
      for rule in var.lifecycle_rules :
      [for selection in rule.selection :
      contains(["imageCountMoreThan", "sinceImagePushed"], selection.count_type)]
    ]))
    error_message = "Valid values for count_type are: imageCountMoreThan or sinceImagePushed."
  }
  validation {
    condition = alltrue([
      for rule in var.lifecycle_rules :
      contains(["expire"], rule.action.type)
    ])
    error_message = "Valid values for action.type are: expire."
  }
}
