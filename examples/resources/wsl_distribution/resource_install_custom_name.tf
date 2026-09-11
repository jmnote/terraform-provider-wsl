# Install mode with a custom name: set name explicitly to register the
# distribution under something other than its Store identifier. Multiple
# instances of the same Store distribution can coexist this way, each
# under its own name.
resource "wsl_distribution" "debian1" {
  name         = "debian1"
  distribution = "Debian"
}

resource "wsl_distribution" "debian2" {
  name         = "debian2"
  distribution = "Debian"
}
