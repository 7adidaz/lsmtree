# data "http" "lsmtree" {
#   url = "http://13.223.248.45/${var.key}"
# }

# resource "local_file" "example" {
#   content  = "this is your value: ${data.http.lsmtree.response_body}"
#   filename = "${path.module}/hello.txt"
# }

# resource "aws_s3_bucket" "data_dump" {
#   bucket = "data-dump-bucket-${data.aws_caller_identity.current.account_id}"
# }

resource "tls_private_key" "lsmtree_key" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "aws_key_pair" "lsmtree_key_pair" {
  key_name   = var.key_name
  public_key = tls_private_key.lsmtree_key.public_key_openssh
}

resource "aws_launch_template" "lsmtree_template" {
  name        = "LSMTree"
  description = "LSM tree KV store: builds from source, runs behind nginx on :80"

  image_id      = var.ami_id
  instance_type = var.instance_type
  key_name      = aws_key_pair.lsmtree_key_pair.key_name

  vpc_security_group_ids = [
    aws_default_security_group.default.id,
    aws_security_group.lsmtree.id,
  ]

  user_data = base64encode(file("${path.module}/user_data.sh"))
}

resource "aws_autoscaling_group" "lsmtree_asg" {
  vpc_zone_identifier = data.aws_subnets.default.ids

  min_size         = 1
  max_size         = 1
  desired_capacity = 1
  launch_template {
    id      = aws_launch_template.lsmtree_template.id
    version = "$Latest"
  }
}

data "aws_instances" "lsmtree_instances" {
  instance_tags = {
    "aws:autoscaling:groupName" = aws_autoscaling_group.lsmtree_asg.name
  }

  instance_state_names = ["running"]
}
