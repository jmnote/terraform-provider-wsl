data "wsl_instance" "ubuntu" {
  name = "Ubuntu"
}

output "ubuntu_version" {
  value = data.wsl_instance.ubuntu.version
}
