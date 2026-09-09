output "private_key" {
  value     = tls_private_key.lsmtree_key.private_key_pem
  sensitive = true
}

output "ssh_command" {
  value = <<-EOF
      tofu output -raw private_key > ~/.ssh/lsmtree-key.pem
      chmod 400 ~/.ssh/lsmtree-key.pem
      ssh -i ~/.ssh/lsmtree-key.pem ec2-user@${try(data.aws_instances.lsmtree_instances.public_ips[0], "<no-running-instance-yet>")}
  EOF
}
