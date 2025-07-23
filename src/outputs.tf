output "registry_id" {
  value       = one(module.ecr[*].registry_id)
  description = "Registry ID"
}

output "repository_name" {
  value       = one(module.ecr[*].repository_name)
  description = "Name of first repository created"
}

output "repository_url" {
  value       = one(module.ecr[*].repository_url)
  description = "URL of first repository created"
}

output "repository_arn" {
  value       = one(module.ecr[*].repository_arn)
  description = "ARN of first repository created"
}

output "ecr_user_name" {
  value       = one(aws_iam_user.ecr[*].name)
  description = "ECR user name"
}

output "ecr_user_arn" {
  value       = one(aws_iam_user.ecr[*].arn)
  description = "ECR user ARN"
}

output "ecr_user_unique_id" {
  value       = one(aws_iam_user.ecr[*].unique_id)
  description = "ECR user unique ID assigned by AWS"
}
