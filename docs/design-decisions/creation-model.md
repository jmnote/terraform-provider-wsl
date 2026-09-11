# Support Both Import and Install Creation Modes

**Summary:** `wsl_distribution` supports two mutually exclusive creation modes -- importing a root filesystem tar (`wsl --import`) and installing a Microsoft Store distribution with no tar file (`wsl --install --distribution`) -- rather than only the former.
**Created**: 2026-09-12
**Author**: [@jmnote](https://github.com/jmnote)

---

## Background

`wsl.exe` offers two families of primitive that can register a new
distribution:

- `wsl --install [--distribution <name>] [--name <custom-name>] [--location <dir>] [--no-launch] [--web-download]`
- `wsl --import <Name> <InstallLocation> <FileName> [--version 1|2]`

The first draft of this decision shipped `wsl --import` only, on the
reasoning that `wsl --install` was too Store/network-dependent and too
inconsistent about custom naming to be a reliable Terraform primitive. That
reasoning about `--name` is still correct (see Decision, below), but
rejecting `--install` entirely does not reflect how WSL is actually used:
the overwhelming majority of real-world WSL distributions are installed via
`wsl --install <Distribution>` (or the interactive Store flow it wraps),
not imported from a hand-built rootfs tar. A provider that could only ever
import a custom tarball would not represent that common, everyday case,
and would force every user to either build their own rootfs archive or use
this provider only for a narrow slice of their WSL usage.

The original requirements explicitly allowed shipping a single creation
mode in v0.1.0 "if both cannot be supported reliably" -- but both turned
out to be supportable, each with the constraints described below.

## Decision

`wsl_distribution` supports **two mutually exclusive creation modes** in
the same resource, selected by which arguments are set:

- **Import mode** (`rootfs` + `location`): `wsl --import`. Bring your own
  root filesystem tar/tar.gz; supports an arbitrary `name`; no Store or
  network dependency.
- **Install mode** (`distribution`): `wsl --install --distribution
  <Distribution> --no-launch`, the exact primitive `wsl --install
  <Distribution>` uses day to day. No tar file needed, and no other
  registered distribution on the host is read or touched.

The `--name` limitation is real and is handled explicitly rather than
ignored: `wsl.exe` does not support installing a Store distribution under
a custom name, so install mode registers the distribution under whatever
`distribution` is, not an arbitrary name Terraform could otherwise choose.
Rather than silently ignoring a user-supplied `name` that would never take
effect, both `internal/wsl/client.go` (`createInstall`) and the resource's
`ValidateConfig` require `name == distribution` in install mode and return
a clear diagnostic explaining why if they differ -- turning the researched
WSL constraint into an honest, actionable error instead of a footgun.

`wsl --install` also has no per-invocation `--version` flag (unlike
`--import`), so install mode applies a requested `version` with a
follow-up `wsl --set-version` call after install completes, rather than
leaving it to the host's mutable `wsl --set-default-version` setting.

## Consequences

- The schema needs mutual-exclusion validation between `rootfs`+`location`
  and `distribution` (`ValidateConfig` at plan time, `internal/wsl/client.go`
  at apply time as a backstop).
- Install mode cannot give the distribution a name Terraform chooses freely;
  `name` must equal `distribution`. Users who need an arbitrary name must
  use import mode.
- Install mode depends on Microsoft Store/network access at apply time,
  which is outside this provider's control and can make `terraform apply`
  less deterministic than import mode; see the "Known limitations" section
  of README.md.
- See [Observable State vs. Creation-Time Attributes](observable-state.md)
  for how the resource handles that neither mode's creation-time inputs
  are recoverable via `Read`.
