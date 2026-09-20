# Releasing

This is a maintainer process; regular contributors won't need it.

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
