/**
We don't include this in the source component, because it requires a specific version of terraform or tofu we'd rather not dictate, this is a nice utility to have
*/

variable "imports" {
  type = object({
    repository = bool
  })
  default = {
    repository = false
  }
}

import {
  for_each = var.imports.repository ? toset([var.repository_name]) : toset([])
  to       = module.ecr.aws_ecr_repository.this[0]
  id       = var.repository_name
}
