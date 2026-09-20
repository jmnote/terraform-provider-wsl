# Install mode: no rootfs tar needed. name is always required -- see
# docs/design/decisions/required-name.md for why this provider does not
# default it to distribution the way wsl.exe itself does.
resource "wsl_instance" "ubuntu" {
  name         = "ubuntu"
  distribution = "Ubuntu-24.04"
}
