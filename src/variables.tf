variable "region" {
  type        = string
  description = "AWS Region"
}

variable "images" {
  type        = list(string)
  description = "List of image names (ECR repo names) to create repos for"
}

variable "image_tag_mutability" {
  type        = string
  description = "The tag mutability setting for the repository. Must be one of: `MUTABLE` or `IMMUTABLE`"
  default     = "MUTABLE"
}

variable "max_image_count" {
  type        = number
  description = "Max number of images to store. Old ones will be deleted to make room for new ones."
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

variable "protected_tags" {
  type        = list(string)
  description = "Tags to refrain from deleting"
  default     = []
}

variable "enable_lifecycle_policy" {
  type        = bool
  description = "Enable/disable image lifecycle policy"
}

variable "principals_lambda" {
  type        = list(string)
  description = "Principal account IDs of Lambdas allowed to consume ECR"
  default     = []
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

variable "custom_lifecycle_rules" {
  description = "Custom lifecycle rules to override or complement the default ones"
  type = list(object({
    description = optional(string)
    selection = object({
      tagStatus      = string
      countType      = string
      countNumber    = number
      countUnit      = optional(string)
      tagPrefixList  = optional(list(string))
      tagPatternList = optional(list(string))
    })
    action = object({
      type = string
    })
  }))
  default = []

  validation {
    condition = alltrue([
      for rule in var.custom_lifecycle_rules :
      rule.selection.tagStatus != "tagged" || (length(coalesce(rule.selection.tagPrefixList, [])) > 0 || length(coalesce(rule.selection.tagPatternList, [])) > 0)
    ])
    error_message = "if tagStatus is tagged - specify tagPrefixList or tagPatternList"
  }
  validation {
    condition = alltrue([
      for rule in var.custom_lifecycle_rules :
      rule.selection.countNumber > 0
    ])
    error_message = "Count number should be > 0"
  }

  validation {
    condition = alltrue([
      for rule in var.custom_lifecycle_rules :
      contains(["tagged", "untagged", "any"], rule.selection.tagStatus)
    ])
    error_message = "Valid values for tagStatus are: tagged, untagged, or any."
  }
  validation {
    condition = alltrue([
      for rule in var.custom_lifecycle_rules :
      contains(["imageCountMoreThan", "sinceImagePushed"], rule.selection.countType)
    ])
    error_message = "Valid values for countType are: imageCountMoreThan or sinceImagePushed."
  }

  validation {
    condition = alltrue([
      for rule in var.custom_lifecycle_rules :
      rule.selection.countType != "sinceImagePushed" || rule.selection.countUnit != null
    ])
    error_message = "For countType = 'sinceImagePushed', countUnit must be specified."
  }
}
