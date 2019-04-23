variable "roles" {
  type = "map"

  default = {}
}

variable "repository_name" {}
variable "project" {}

variable "environment" {
  default = "dev"
}

variable "profile" {
  default = ""
}

variable "region" {
  default = "ca-central-1"
}

variable "account_id" {
  default = ""
}
