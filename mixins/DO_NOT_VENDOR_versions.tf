# DO NOT VENDOR THIS FILE
# This file is used to satisfy the tests of this repository and should not be vendored.
terraform {
  required_version = ">= 1.0.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 4.9.0"
    }
  }
}
