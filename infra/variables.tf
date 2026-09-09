variable "key" {
  default = "roro"
  type    = string
}

variable "region" {
  default = "us-east-1"
  type    = string
}

// launch template variables
variable "ami_id" {
  description = "Amazon Linux 2023 AMI for the launch template"
  default     = "ami-0bdc7d025135d7b49"
  type        = string
}

variable "instance_type" {
  default = "t3.micro"
  type    = string
}

variable "key_name" {
  description = "Name of the EC2 key pair created from the generated TLS private key"
  default     = "lsmtree-key"
  type        = string
}

variable "vpc_id" {
  description = "Default VPC; both imported security groups live here"
  default     = "vpc-054604a31d88867b8"
  type        = string
}
