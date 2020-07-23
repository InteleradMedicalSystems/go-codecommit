output "clone_url_http" {
  value = aws_codecommit_repository.repo.clone_url_http
}

output "account_id" {
  value = data.aws_caller_identity.current.account_id
}

