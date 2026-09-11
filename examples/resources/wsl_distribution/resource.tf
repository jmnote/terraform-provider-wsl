resource "wsl_distribution" "worker" {
  name     = "worker"
  rootfs   = "C:\\images\\ubuntu-24.04.tar"
  location = "D:\\WSL\\worker"
  version  = 2
}
