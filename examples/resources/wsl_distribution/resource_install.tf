# Install mode: no rootfs tar needed. wsl.exe registers a Store
# distribution under its own identifier, so name must equal distribution.
resource "wsl_distribution" "ubuntu" {
  name         = "Ubuntu-24.04"
  distribution = "Ubuntu-24.04"
}
