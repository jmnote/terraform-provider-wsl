resource "wsl_distribution" "debian3" {
  name     = "debian3"
  rootfs   = "C:\\images\\debian-12.tar"
  location = "D:\\WSL\\debian3"
  version  = 2
}
