# Use the Instance's Registration Name as Resource Identity

**Summary:** `wsl_instance` uses the instance's registration name as its `name` attribute and its entire classic, ID-based `terraform import` identity, rather than adopting Terraform 1.12's newer resource identity schema feature.

---

## Background

Terraform resource identity needs to be chosen carefully: it is what
`terraform import wsl_instance.debian3 <id>` addresses, and what ties
a `wsl_instance` resource to a real WSL instance across plans.

The instance's registration name (`debian3`, `Ubuntu-24.04`, ...) is
the only stable, human-meaningful handle `wsl.exe` exposes for a
distribution, and it is exactly what every `wsl` subcommand this provider
calls (`--import`, `--install`, `--set-version`, `--unregister`,
`--list --verbose`) addresses distributions by.

Terraform 1.12 introduced an additional, optional "resource identity"
schema feature: a second, restricted data object stored alongside resource
state, intended for more precise cross-state matching.

## Decision

The registration name is used as the resource's `name` attribute and, via
`resource.ImportStatePassthroughID`, as the entire
`terraform import wsl_instance.debian3 debian3` identity -- the classic,
ID-based import mechanism.

v0.1.0 does not adopt Terraform 1.12's newer resource identity schema
feature: it requires a newer Terraform CLI than this provider otherwise
needs, and the classic ID-based import already maps perfectly onto WSL's
one true identifier.
