data "wsl_distribution" "ubuntu" {
  name = "Ubuntu"
}

output "ubuntu_version" {
  value = data.wsl_distribution.ubuntu.version
}
