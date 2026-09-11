# Use the Distribution's Registration Name as Resource Identity

**Summary:** `wsl_distribution` uses the distribution's registration name as its `name` attribute and its entire classic, ID-based `terraform import` identity, rather than adopting Terraform 1.12's newer resource identity schema feature.
**Created**: 2026-09-12
**Author**: [@jmnote](https://github.com/jmnote)

---

## Background

Terraform resource identity needs to be chosen carefully: it is what
`terraform import wsl_distribution.worker <id>` addresses, and what ties
a `wsl_distribution` resource to a real WSL distribution across plans.

The distribution's registration name (`worker`, `Ubuntu-24.04`, ...) is
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
`terraform import wsl_distribution.worker worker` identity -- the classic,
ID-based import mechanism.

v0.1.0 does not adopt Terraform 1.12's newer resource identity schema
feature: it requires a newer Terraform CLI than this provider otherwise
needs, and the classic ID-based import already maps perfectly onto WSL's
one true identifier.

## Consequences

- This provider has no minimum Terraform CLI version requirement beyond
  what the Plugin Framework itself needs.
- Adopting the newer identity schema in addition to the classic `id` is a
  plausible, low-risk v0.2+ enhancement once the provider has real-world
  usage to justify the extra schema surface.
