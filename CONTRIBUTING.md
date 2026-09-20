# Contributing

Requires [Go](https://go.dev) 1.23+ and PowerShell.

```powershell
.\build.ps1          # fmt, vet, test, build
.\build.ps1 test     # just the unit tests
.\build.ps1 build
```

[`build.ps1`](build.ps1) needs nothing beyond PowerShell and a Go
toolchain already on PATH -- no `make`, no separate installer -- since
this provider only ever builds/tests on native Windows anyway (see
[`docs/design-decisions/platform-support.md`](docs/design-decisions/platform-support.md)).
Its tasks are thin wrappers; run `go build ./...`, `go test ./...`,
`go vet ./...`, or `gofmt -l -w .` directly if you'd rather skip the
script entirely.

Regenerate documentation under `docs/` after changing any schema
(requires [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs)):

```powershell
.\build.ps1 generate   # or: go generate ./...
```

## Acceptance tests

Acceptance tests exercise real Terraform apply/destroy cycles and require
an actual Windows host with WSL installed. There are two, matching the two
creation modes, each opt-in via its own environment variable so `go test`
never creates a real distribution by accident:

```bash
# Install mode: no tar needed, but needs Microsoft Store/network access
# and a distribution identifier not already registered on the host.
TF_ACC=1 WSL_ACC_TEST_DISTRIBUTION=Ubuntu-24.04 go test ./internal/provider/... -v -run TestAccWSLDistributionResource_InstallMode

# Import mode: needs a small root filesystem tar (does not need to be
# bootable -- this test never launches the distribution).
TF_ACC=1 WSL_ACC_TEST_ROOTFS=C:\path\to\rootfs.tar go test ./internal/provider/... -v -run TestAccWSLDistributionResource_ImportMode
```

[`build.ps1`](build.ps1)'s `testacc` task is a shorthand for the same
thing (it sets `TF_ACC=1` and runs `go test` against
`./internal/provider/...` for you); you still need to set
`WSL_ACC_TEST_DISTRIBUTION` or `WSL_ACC_TEST_ROOTFS` yourself first, e.g.
`$env:WSL_ACC_TEST_DISTRIBUTION = "Ubuntu-24.04"; .\build.ps1 testacc`.

They are not run automatically in CI (see
[`.github/workflows/test.yml`](.github/workflows/test.yml)); run them
locally on a machine where creating and destroying real WSL distributions
is acceptable.
