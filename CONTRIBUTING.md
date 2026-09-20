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
[`docs/design/decisions/platform-support.md`](docs/design/decisions/platform-support.md)).
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
never creates a real instance by accident:

```bash
# Install mode: no tar needed, but needs Microsoft Store/network access
# and a distribution identifier not already registered on the host.
TF_ACC=1 WSL_ACC_TEST_DISTRIBUTION=Ubuntu-24.04 go test ./internal/provider/... -v -run TestAccWSLInstanceResource_InstallMode

# Import mode: needs a small root filesystem tar (does not need to be
# bootable -- this test never launches the instance).
TF_ACC=1 WSL_ACC_TEST_ROOTFS=C:\path\to\rootfs.tar go test ./internal/provider/... -v -run TestAccWSLInstanceResource_ImportMode
```

[`build.ps1`](build.ps1)'s `testacc` task is a shorthand for the same
thing (it sets `TF_ACC=1`, defaults `TF_LOG` to `DEBUG` if not already set
-- so wsl.exe's install/import progress is visible instead of a long
silence -- and runs `go test` against `./internal/provider/...` for you);
you still need to set `WSL_ACC_TEST_DISTRIBUTION` or `WSL_ACC_TEST_ROOTFS`
yourself first, e.g.
`$env:WSL_ACC_TEST_DISTRIBUTION = "Ubuntu-24.04"; .\build.ps1 testacc`.

They are not run automatically in CI (see
[`.github/workflows/test.yml`](.github/workflows/test.yml)); run them
locally on a machine where creating and destroying real WSL instances
is acceptable.

## Testing against a local build

To drive a locally built provider with real `terraform plan`/`apply`
commands (rather than the Go-level acceptance tests above), use
Terraform's own [development overrides](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers)
feature -- a one-time setup, not something `build.ps1` wraps, since once
it's in place plain `terraform plan`/`apply` (no `terraform init` needed)
already do exactly this:

1. `.\build.ps1 install` (or `go install ./...`) builds and installs the
   provider to `terraform-provider-wsl.exe` under `$(go env GOPATH)\bin`.
2. Point Terraform at that directory instead of the registry, by creating
   a CLI config file (anywhere; this example keeps it out of the repo)
   containing:

   ```hcl
   provider_installation {
     dev_overrides {
       "jmnote/wsl" = "C:\\Users\\<you>\\go\\bin"
     }
     direct {}
   }
   ```

   and pointing the `TF_CLI_CONFIG_FILE` environment variable at it for
   your session, e.g. `$env:TF_CLI_CONFIG_FILE = "C:\path\to\dev.tfrc"`.
3. `cd examples/provider-install-verification` and run `terraform plan`.
   Terraform prints a "provider development overrides are in effect"
   warning to confirm the local build (not the registry) is the one
   responding. Only run `plan` there, never `apply` -- see that file's
   header comment for why.
