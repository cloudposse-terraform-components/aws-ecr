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
  description = "The tag mutability setting for the repository. Must be one of: `MUTABLE`, `IMMUTABLE`, `IMMUTABLE_WITH_EXCLUSION`, or `MUTABLE_WITH_EXCLUSION`"
  default     = "MUTABLE"
}

variable "image_tag_mutability_exclusion_filter" {
  type = list(object({
    filter      = string
    filter_type = optional(string, "WILDCARD")
  }))
  default     = []
  description = "List of exclusion filters for image tag mutability. Each filter object must contain 'filter' and 'filter_type' attributes. Requires AWS provider >= 6.8.0"

  validation {
    condition = alltrue([
      for filter in var.image_tag_mutability_exclusion_filter :
      contains(["WILDCARD"], filter.filter_type)
    ])
    error_message = "filter_type must be `WILDCARD`"
  }

  validation {
    condition = alltrue([
      for filter in var.image_tag_mutability_exclusion_filter :
      length(trimspace(filter.filter)) > 0
    ])
    error_message = "filter value cannot be empty or contain only whitespace."
  }
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

variable "protected_tags_keep_count" {
  type        = number
  description = "Number of Image versions to keep for protected tags"
  default     = 999999
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
  description = "Custom lifecycle rules to override or complement the default ones. Action type can be 'expire' or 'transition'. Use 'transition' with targetStorageClass='archive' to archive images instead of deleting them. StorageClass can be 'standard' or 'archive' and is omitted from the rendered policy when not set."
  type = list(object({
    description = optional(string)
    selection = object({
      tagStatus      = string
      storageClass   = optional(string)
      countType      = string
      countNumber    = number
      countUnit      = optional(string)
      tagPrefixList  = optional(list(string))
      tagPatternList = optional(list(string))
    })
    action = object({
      type               = string
      targetStorageClass = optional(string)
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
      contains(["imageCountMoreThan", "sinceImagePushed", "sinceImageTransitioned"], rule.selection.countType)
    ])
    error_message = "Valid values for countType are: imageCountMoreThan, sinceImagePushed, or sinceImageTransitioned."
  }

  validation {
    condition = alltrue([
      for rule in var.custom_lifecycle_rules :
      !contains(["sinceImagePushed", "sinceImageTransitioned"], rule.selection.countType) || rule.selection.countUnit != null
    ])
    error_message = "For countType = 'sinceImagePushed' or 'sinceImageTransitioned', countUnit must be specified."
  }

  validation {
    condition = alltrue([
      for rule in var.custom_lifecycle_rules :
      contains(["expire", "transition"], rule.action.type)
    ])
    error_message = "Valid values for action.type are: expire or transition."
  }

  validation {
    condition = alltrue([
      for rule in var.custom_lifecycle_rules :
      rule.action.type != "transition" || rule.action.targetStorageClass != null
    ])
    error_message = "For action.type = 'transition', targetStorageClass must be specified."
  }

  validation {
    condition = alltrue([
      for rule in var.custom_lifecycle_rules :
      rule.selection.storageClass == null || contains(["standard", "archive"], rule.selection.storageClass)
    ])
    error_message = "Valid values for storageClass are: standard or archive. Omit to not include storageClass in the rendered policy."
  }

  # ECR requires countType=sinceImageTransitioned when selection.storageClass=archive,
  # and rejects that countType otherwise. Catch it here for a clearer error than
  # ECR's 400 InvalidParameterException.
  validation {
    condition = alltrue([
      for rule in var.custom_lifecycle_rules :
      rule.selection.storageClass != "archive" || rule.selection.countType == "sinceImageTransitioned"
    ])
    error_message = "When selection.storageClass is 'archive', countType must be 'sinceImageTransitioned' (ECR does not allow imageCountMoreThan/sinceImagePushed with the archive storage class)."
  }

  validation {
    condition = alltrue([
      for rule in var.custom_lifecycle_rules :
      rule.selection.countType != "sinceImageTransitioned" || rule.selection.storageClass == "archive"
    ])
    error_message = "countType 'sinceImageTransitioned' is only valid when selection.storageClass is 'archive'."
  }
}

variable "default_lifecycle_rules_settings" {
  description = "Default lifecycle rules settings"
  type = object({
    untagged_image_rule = optional(object({
      enabled = optional(bool, true)
      }), {
      enabled = true
    })
    remove_old_image_rule = optional(object({
      enabled = optional(bool, true)
      }), {
      enabled = true
    })
  })
  default = {
    untagged_image_rule = {
      enabled = true
    }
    remove_old_image_rule = {
      enabled = true
    }
  }
}
