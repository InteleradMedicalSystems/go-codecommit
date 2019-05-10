data "aws_caller_identity" "current" {}

resource "aws_codecommit_repository" "repo" {
  repository_name = "${var.repository_name}"
  description     = "Test codecommit credential helper"
  default_branch  = "master"
}

provider "aws" {
  region  = "${var.region}"
  version = "~> 1.37"

  #assume_role {
  #  role_arn = "${lookup(var.roles, var.environment)}"
  #}
}

terraform = {
  required_version = ">=0.11.7"
}
