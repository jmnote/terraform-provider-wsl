# Target Terraform Running Natively on Windows Only

**Summary:** v0.1.0 supports Terraform running on `windows_amd64`/`windows_arm64` and shelling out to `wsl.exe` directly; running Terraform inside a WSL distribution and calling `/mnt/c/Windows/System32/wsl.exe` across the interop boundary is out of scope.

---

## Background

The canonical way to run WSL from automation is `wsl.exe`, a Windows
binary. Terraform itself can run either natively on Windows, or inside a
WSL distribution (calling out across the Windows/WSL interop boundary to
`/mnt/c/Windows/System32/wsl.exe`).

## Decision

The execution model this provider targets is Terraform running natively on
Windows and shelling out to `wsl.exe` directly:

```text
Windows

terraform.exe
    |
    +-- terraform-provider-wsl.exe
            |
            +-- wsl.exe
                    |
                    +-- Ubuntu
                    +-- debian1
                    +-- debian2
```

Running Terraform inside a WSL distribution and calling
`/mnt/c/Windows/System32/wsl.exe` is not supported in v0.1.0: it adds path
translation and encoding complexity across the Windows/WSL interop
boundary that is out of scope for this release.
`internal/provider.Configure` rejects that case (and any non-Windows host)
with an explicit diagnostic rather than failing in a confusing way later.

v0.1.0 therefore supports Terraform running on `windows_amd64` and
`windows_arm64` only; `.goreleaser.yml`'s build matrix is restricted to
`windows/amd64,windows/arm64` to match.

## Consequences

- Users running Terraform from inside a WSL distribution cannot use this
  provider in v0.1.0; they must run `terraform apply` from Windows
  PowerShell/cmd instead.
- The release matrix, CI runner (`windows-latest`), and provider
  Configure logic all need to stay consistent with this decision; a
  contributor adding a new OS/arch to one without the others is a bug.
- WSL-inside-WSL-interop support is a plausible v0.2+ direction once the
  encoding/path-translation complexity it requires is worth taking on.
