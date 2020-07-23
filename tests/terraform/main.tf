data "aws_caller_identity" "current" {
}

resource "aws_codecommit_repository" "repo" {
  repository_name = var.repository_name
  description     = "Test CodeCommit credential helper"
  default_branch  = "master"
}

provider "aws" {
  region = var.region
}

