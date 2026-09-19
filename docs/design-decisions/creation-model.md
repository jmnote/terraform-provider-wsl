# Support Both Install and Import Creation Modes

**Summary:** `wsl_distribution` supports two mutually exclusive creation modes -- installing a Microsoft Store distribution with no tar file (`wsl --install`) and importing your own root filesystem tar or `.wsl` file (`wsl --import`) -- with an optional custom `name` available in both.

---

## Background

`wsl.exe` offers two families of primitive that can register a new
distribution:

- `wsl --install <Distro> [--name <Name>] [--location <dir>] [--no-launch] [--web-download]`
- `wsl --import <Name> <InstallLocation> <FileName> [--version 1|2]`

The first draft of this decision shipped `wsl --import` only, on the
reasoning that `wsl --install` was too Store/network-dependent and too
inconsistent about custom naming to be a reliable Terraform primitive.
Rejecting `--install` entirely did not reflect how WSL is actually used,
though: the overwhelming majority of real-world WSL distributions are
installed via `wsl --install <Distribution>` (or the interactive Store
flow it wraps), not imported from a hand-built rootfs tar. A provider that
could only ever import a custom tarball would not represent that common,
everyday case, and would force every user to either build their own
rootfs archive or use this provider only for a narrow slice of their WSL
usage.

A second draft added install mode but still required `name` to equal
`distribution`, on research (a community report of `wsl --install --name`
being rejected for "legacy" distributions) that turned out to be outdated.
Two things corrected it, verified against a real, current WSL host
(2.7.13.0):

- `wsl --install <Distro> [Options...]` takes the distribution
  **positionally**, not via a `--distribution` flag as originally
  implemented -- confirmed by a real, successful
  `wsl --install ArchLinux --name a1 --no-launch`.
- `--name` **is** a working `--install` option on current wsl.exe, with no
  "legacy distribution" restriction: the same real install registered the
  distribution under the custom name `a1`, not `ArchLinux`. Current
  `wsl --help` also lists `--name <Name>` as a plain option under
  `--install`, and Microsoft's own custom-distro documentation describes
  it as able to override a distribution's default registered name.

The original requirements explicitly allowed shipping a single creation
mode in v0.1.0 "if both cannot be supported reliably" -- but both turned
out to be supportable, each with the constraints described below.

## Decision

`wsl_distribution` supports **two mutually exclusive creation modes** in
the same resource, selected by which arguments are set:

- **Install mode** (`distribution`): `wsl --install <Distribution>
  --no-launch`, the exact primitive `wsl --install <Distribution>` uses
  day to day. No tar file needed, and no other registered distribution on
  the host is read or touched.
- **Import mode** (`rootfs` + `location`): `wsl --import`. Bring your own
  root filesystem tar/tar.gz -- a `.wsl` file (a tar archive with a `.wsl`
  extension, the format Microsoft's own custom-distro tooling and the
  Store both use) works here too, verified with a real
  `wsl --import <Name> <Location> <file>.wsl`, so no separate attribute
  for it is needed.

`name` is optional in both modes. In install mode, an omitted `name`
defaults to `distribution` (wsl.exe's own default when `--name` is not
passed); setting it explicitly registers the distribution under a
different name than its Store identifier, which lets multiple instances
of the same Store distribution coexist under different names. In import
mode, `name` has no such default (there's nothing to default it to) and
must be set explicitly.

`wsl --install` also has no per-invocation `--version` flag (unlike
`--import`), so install mode applies a requested `version` with a
follow-up `wsl --set-version` call after install completes, rather than
leaving it to the host's mutable `wsl --set-default-version` setting.

## Consequences

- The schema needs mutual-exclusion validation between `distribution` and
  `rootfs`+`location` (`ValidateConfig` at plan time, `internal/wsl/client.go`
  at apply time as a backstop), plus a check that `name` is set in import
  mode specifically (no default there).
- Because `name` can now be Computed (defaulted from `distribution`), the
  resource's `Create` must resolve it to a known value itself before
  returning state -- Terraform requires every attribute to be known after
  apply, and `Read`/`refresh` needs a concrete name to look the
  distribution up by regardless of which mode set it.
- Install mode depends on Microsoft Store/network access at apply time,
  which is outside this provider's control and can make `terraform apply`
  less deterministic than import mode; see the "Known limitations" section
  of README.md.
- Applying a requested `version` in install mode is two wsl.exe calls, not
  one (`wsl --install` then `wsl --set-version`), so `Create` can fail
  between them: the distribution now exists but not at the requested
  version. `createInstall` treats this as all-or-nothing and rolls the
  distribution back (`wsl --unregister`) rather than leaving an orphan
  invisible to Terraform state; see `internal/wsl/client.go`.
- Terraform runs multiple resources' CRUD concurrently. `internal/wsl`'s
  `client` serializes state-mutating calls (`Create`, `SetVersion`,
  `Delete`) and excludes concurrent `List`/`Get` snapshots with an
  `RWMutex`, since WSL's distribution registration internals are not
  documented as safe for concurrent registration/unregistration or
  read-during-write access. Independent read snapshots may still run in
  parallel.
- See [Observable State vs. Creation-Time Attributes](observable-state.md)
  for how the resource handles that neither mode's creation-time inputs
  are recoverable via `Read`.
