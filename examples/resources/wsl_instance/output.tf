# Terraform does not print a resource's attributes after apply on its own;
# an explicit output is how to see them, e.g. a ready-to-run connect
# command, without a separate `terraform show`/`state show` step.
output "ubuntu" {
  value = <<-EOT
    Name: ${wsl_instance.ubuntu.name}
    State: ${wsl_instance.ubuntu.state}
    WSL Version: ${wsl_instance.ubuntu.version}

    Connect with: wsl -d ${wsl_instance.ubuntu.name}
  EOT
}
