variable "roles" {
  type = map(string)

  default = {}
}

variable "repository_name" {
}

variable "project" {
}

variable "environment" {
  default = "dev"
}

variable "profile" {
  default = ""
}

variable "account_id" {
  default = ""
}

variable "region" {
  default = "us-east-1"
}

