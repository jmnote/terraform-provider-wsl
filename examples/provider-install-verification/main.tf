####################################################################
# This config is for use with `dev_overrides` (see CONTRIBUTING.md,
# "Testing against a local build") to verify a locally built provider
# actually loads and responds -- not a usage example. See examples/
# for those.
#
# Only run `terraform plan` here, never `apply`: planning a resource
# that doesn't exist in state yet only exercises the provider's
# schema/validation logic and never touches wsl.exe, so there's nothing
# here that needs to already exist on your machine. Applying would
# really install this instance.
####################################################################

terraform {
  required_providers {
    wsl = {
      source = "jmnote/wsl"
    }
  }
}

provider "wsl" {}

resource "wsl_instance" "example" {
  name         = "provider-install-verification"
  distribution = "Ubuntu-24.04"
}
