# terraform-provider-wsl

A Terraform provider for declaratively managing Windows Subsystem for Linux
(WSL) distributions.

```hcl
terraform {
  required_providers {
    wsl = {
      source  = "jmnote/wsl"
      version = "~> 0.1"
    }
  }
}

resource "wsl_distribution" "worker" {
  name     = "worker"
  rootfs   = "C:\\images\\ubuntu-24.04.tar"
  location = "D:\\WSL\\worker"
  version  = 2
}

# ...or install a Microsoft Store distribution directly, no tar needed:
resource "wsl_distribution" "ubuntu" {
  name         = "Ubuntu-24.04"
  distribution = "Ubuntu-24.04"
}
```

## Scope

This provider manages the **lifecycle of a WSL distribution's
registration**: create (either by importing your own root filesystem tar,
or by installing a Microsoft Store distribution with no tar file at all),
read, an in-place WSL-version update, delete, and import. It does **not**
manage anything inside a distribution:

- apt/dnf/apk package management
- arbitrary command execution
- file management
- systemd service management
- SSH management
- Ansible- or Nix-like provisioning
- Kubernetes/k0s-specific functionality

Use cloud-init, Ansible, Nix, or another configuration-management tool for
what happens inside a distribution. This provider is only the local
infrastructure lifecycle layer underneath that.

## Requirements

- Windows (`windows_amd64` or `windows_arm64`). This provider only runs as
  part of Terraform executing natively on Windows and shelling out to
  `wsl.exe`; running Terraform inside a WSL distribution and calling
  `/mnt/c/Windows/System32/wsl.exe` is not supported in v0.1.0.
- WSL installed, with `wsl.exe` available (on PATH by default, or pointed
  to explicitly via the provider's `executable` argument).
- Terraform (a recent 1.x release) and [Go](https://go.dev) 1.23+ if
  building from source.

## Installation

```hcl
terraform {
  required_providers {
    wsl = {
      source  = "jmnote/wsl"
      version = "~> 0.1"
    }
  }
}
```

## Usage

```hcl
provider "wsl" {}

resource "wsl_distribution" "worker" {
  name     = "worker"
  rootfs   = "C:\\images\\ubuntu-24.04.tar"
  location = "D:\\WSL\\worker"
  version  = 2
}

data "wsl_distribution" "ubuntu" {
  name = "Ubuntu"
}
```

`rootfs`+`location` (import mode) and `distribution` (install mode) are
mutually exclusive; set exactly one. See
[`docs/design-decisions/creation-model.md`](docs/design-decisions/creation-model.md)
for why both exist and what each requires.

More examples live under [`examples/`](examples/), and full attribute
reference docs are generated under [`docs/`](docs/).

## Import

```bash
terraform import wsl_distribution.worker worker
```

The distribution's registration name is the entire import identity. WSL
exposes no way to read back the original `rootfs`/`location` or
`distribution` for an existing distribution, so all three are left unset by
import; supply matching values in your configuration afterward. See
[`docs/design-decisions/observable-state.md`](docs/design-decisions/observable-state.md)
for why, and how the schema avoids proposing a destructive replace on that
first `terraform plan` after import.

## ⚠️ Destructive operation warning

`terraform destroy`, and any change to `wsl_distribution`'s `name`,
`rootfs`, `location`, or `distribution`, unregisters the distribution via
`wsl --unregister`, which **permanently deletes its virtual disk and
everything inside it**. There is no undo, and this provider does not take
backups or add any safety net beyond what `terraform plan` already shows
you.

## Platform support

v0.1.0 supports Terraform running on `windows_amd64` and `windows_arm64`
only. See [`docs/design-decisions/platform-support.md`](docs/design-decisions/platform-support.md).
Every v0.1.0 design decision (creation model, resource identity,
Update vs. Replace, locale-independent parsing, platform support) has its
own record under [`docs/design-decisions/`](docs/design-decisions/).

## Known limitations

- **Zero registered distributions.** `wsl --list --verbose` exits non-zero
  with a localized message, and no table to parse, when a host has no WSL
  distributions registered at all. There is currently no
  locale-independent way to distinguish that from a real failure (missing
  `wsl.exe`, WSL not installed); see
  [microsoft/WSL#6235](https://github.com/microsoft/WSL/issues/6235). This
  provider makes a best effort (an empty parse result is treated as an
  empty list) but cannot guarantee correctness on every locale in this
  specific case.
- **`rootfs`/`location`/`distribution` are unrecoverable on import.** See
  "Import" above.
- **No machine-readable WSL CLI output.** All state reconciliation is
  parsed from `wsl --list --verbose`'s human-oriented table; see
  `internal/wsl/parser.go` and
  [`docs/design-decisions/locale-independent-parsing.md`](docs/design-decisions/locale-independent-parsing.md)
  for how that parsing stays locale-independent.
- **Install mode cannot choose a custom registration name.** `wsl.exe`
  registers a Store distribution under its own identifier, so `name` must
  equal `distribution` in install mode; the provider rejects a mismatch
  with a clear error rather than silently ignoring it. Use import mode if
  you need an arbitrary name.
- **Install mode needs Microsoft Store/network access** at apply time,
  which is outside this provider's control and can make `terraform apply`
  less deterministic than import mode; see
  [`docs/design-decisions/creation-model.md`](docs/design-decisions/creation-model.md).

## Development

```bash
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

Regenerate documentation under `docs/` after changing any schema
(requires [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs)):

```bash
go generate ./...
```

### Acceptance tests

Acceptance tests exercise real Terraform apply/destroy cycles and require
an actual Windows host with WSL installed. There are two, matching the two
creation modes, each opt-in via its own environment variable so `go test`
never creates a real distribution by accident:

```bash
# Import mode: needs a small root filesystem tar (does not need to be
# bootable -- this test never launches the distribution).
TF_ACC=1 WSL_ACC_TEST_ROOTFS=C:\path\to\rootfs.tar go test ./internal/provider/... -v -run TestAccWSLDistributionResource_ImportMode

# Install mode: no tar needed, but needs Microsoft Store/network access
# and a distribution identifier not already registered on the host.
TF_ACC=1 WSL_ACC_TEST_DISTRIBUTION=Ubuntu-24.04 go test ./internal/provider/... -v -run TestAccWSLDistributionResource_InstallMode
```

They are not run automatically in CI (see
[`.github/workflows/test.yml`](.github/workflows/test.yml)); run them
locally on a machine where creating and destroying real WSL distributions
is acceptable.

## Releasing

Tagged `v*` pushes run [GoReleaser](https://goreleaser.com) via
[`.github/workflows/release.yml`](.github/workflows/release.yml), building
`windows_amd64`/`windows_arm64` binaries, `SHA256SUMS`, a GPG signature over
the checksum file, and a GitHub Release, matching the
[Terraform Registry publishing requirements](https://developer.hashicorp.com/terraform/registry/providers/publishing).
This needs the following repository secrets, which are not configured by
this codebase and must be set by a maintainer with a real GPG key before
cutting a release:

| Secret             | Purpose                                                  |
| ------------------ | --------------------------------------------------------- |
| `GPG_PRIVATE_KEY`  | ASCII-armored private key used to sign `SHA256SUMS`      |
| `GPG_PASSPHRASE`   | Passphrase for the above key (omit if the key has none)  |

The corresponding public key must also be uploaded to the provider's
signing key settings in the Terraform Registry. No release is published by
this repository's automation until a maintainer configures these secrets.

## License

[Apache License 2.0](LICENSE).
