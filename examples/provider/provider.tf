terraform {
  required_providers {
    wsl = {
      source  = "jmnote/wsl"
      version = "~> 0.1"
    }
  }
}

provider "wsl" {
  # Optional: override the wsl.exe executable this provider invokes.
  # Defaults to "wsl.exe", resolved via PATH.
  # executable = "wsl.exe"
}
