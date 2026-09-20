---
page_title: "Getting Started with the WSL Provider"
subcategory: ""
description: |-
  A first walkthrough: install a WSL distribution with Terraform.
---

# Getting Started with the WSL Provider

This walks through installing a single WSL distribution with Terraform,
using the Microsoft Store install mode (no rootfs tar needed).

## 1. Configure the provider

```terraform
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
```

## 2. Declare a distribution

```terraform
# Install mode: no rootfs tar needed. name is optional here and defaults
# to distribution (wsl.exe's own default when --name is not passed).
resource "wsl_distribution" "ubuntu" {
  distribution = "Ubuntu-24.04"
}
```

`distribution` values come from WSL's own catalog (`wsl --list --online`).
A bare flavor name like `"Ubuntu"` floats to whatever that flavor's catalog
entry currently marks as default; a specific name like `"Ubuntu-24.04"`
pins to that entry.

## 3. Apply

```shell
terraform init
terraform apply
```

This runs `wsl --install Ubuntu-24.04 --no-launch`, so Terraform can
create the registration without waiting on first-launch setup.

## Next steps

- Bringing your own rootfs tar instead of a Store distribution: see
  [import mode](../resources/distribution.md#import-mode-bring-your-own-rootfs-tar-or-a-wsl-file)
  in the `wsl_distribution` resource documentation.
- Full attribute reference: [`wsl_distribution`](../resources/distribution.md).
- `terraform destroy`, or changing `name`/`distribution`/`rootfs`/`location`,
  permanently deletes the distribution's virtual disk; see the resource
  documentation's destructive-delete note before you apply.
