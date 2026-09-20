---
page_title: "Getting Started with the WSL Provider"
subcategory: ""
description: |-
  A first walkthrough: install a WSL instance with Terraform.
---

# Getting Started with the WSL Provider

This walks through installing a single WSL instance with Terraform,
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

## 2. Declare an instance

```terraform
# Install mode: no rootfs tar needed. name is always required -- see
# docs/design/decisions/required-name.md for why this provider does not
# default it to distribution the way wsl.exe itself does.
resource "wsl_instance" "ubuntu" {
  name         = "ubuntu"
  distribution = "Ubuntu-24.04"
}
```

`distribution` values come from WSL's own catalog (`wsl --list --online`).
A bare flavor name like `"Ubuntu"` floats to whatever that flavor's catalog
entry currently marks as default; a specific name like `"Ubuntu-24.04"`
pins to that entry. `name` is a separate, always-required value: see
[required-name.md](https://github.com/jmnote/terraform-provider-wsl/blob/main/docs/design/decisions/required-name.md)
for why it is never defaulted from `distribution`.

## 3. Apply

```shell
terraform init
terraform apply
```

This runs `wsl --install Ubuntu-24.04 --name ubuntu --no-launch`, so
Terraform can create the registration without waiting on first-launch
setup.

## 4. See what you created

`terraform apply` itself only reports create/change/destroy counts, not a
resource's attribute values. To see those -- e.g. a ready-to-run connect
command -- declare an output:

```terraform
# Terraform does not print a resource's attributes after apply on its own;
# an explicit output is how to see them, e.g. a ready-to-run connect
# command, without a separate `terraform show`/`state show` step.
output "ubuntu" {
  value = <<-EOT
    Name: ${wsl_instance.ubuntu.name}
    State: ${wsl_instance.ubuntu.state}
    WSL Version: ${wsl_instance.ubuntu.version}

    Connect with: wsl -d ${wsl_instance.ubuntu.name}
  EOT
}
```

`terraform apply` then prints it under `Outputs:`, and `terraform output
ubuntu` reprints it any time afterward without another apply.

## Next steps

- Bringing your own rootfs tar instead of a Store distribution: see
  [import mode](../resources/instance.md#import-mode-bring-your-own-rootfs-tar-or-a-wsl-file)
  in the `wsl_instance` resource documentation.
- Full attribute reference: [`wsl_instance`](../resources/instance.md).
- `terraform destroy`, or changing `name`/`distribution`/`rootfs`/`location`,
  permanently deletes the instance's virtual disk; see the resource
  documentation's destructive-delete note before you apply.
