# Multiple instances can be stamped out from the same Store distribution,
# each under its own name.
resource "wsl_instance" "debian1" {
  name         = "debian1"
  distribution = "Debian"
}

resource "wsl_instance" "debian2" {
  name         = "debian2"
  distribution = "Debian"
}
