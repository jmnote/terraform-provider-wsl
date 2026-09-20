# Require an Explicit `name` in Every Creation Mode

**Summary:** `name` is `Required` in both install and import mode; it is
never defaulted from `distribution`.

---

## Background

`wsl.exe` itself makes `--name` optional on `wsl --install`, defaulting to
`distribution` when omitted. But values like `"Ubuntu"` or `"Debian"`
(typical `distribution` values) are exactly the names people are most
likely to already have registered for unrelated, real work -- a
daily-driver distribution, one set up for an editor's remote-WSL
integration. Following `wsl.exe`'s own default would let a config
silently target that same name.

Unlike a cloud resource, a WSL instance carries no tag or label --
`wsl --list --verbose` exposes nothing but the registration name itself.
There is no secondary signal to notice a collision after the fact, or to
tell "Terraform's" instance apart from one set up by hand; the name *is*
the only identity there is.

## Decision

`name` is `Required` in the schema, in both creation modes, with no
default. Install mode always needs both `name` and `distribution`; see
`examples/resources/wsl_instance/resource.tf`.
