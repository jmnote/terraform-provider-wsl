# Install mode: no rootfs tar needed. name is optional here and defaults
# to distribution (wsl.exe's own default when --name is not passed).
resource "wsl_distribution" "ubuntu" {
  distribution = "Ubuntu-24.04"
}
