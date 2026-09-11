# Never Guess Creation-Time Attributes That WSL Cannot Report Back

**Summary:** `distribution`, `rootfs`, and `location` are creation-time-only, `Optional + Computed` attributes that `Read` never fabricates a value for; a custom plan modifier lets a freshly-imported resource adopt a configured value without Terraform proposing a destructive replace.

---

## Background

`wsl --list --verbose` is the only broadly available way to query
registered distributions, and it reliably reports exactly three things:
name, run state, and WSL version. It cannot report:

- whether it was originally created via install or import mode (i.e. the
  `distribution` value, if any; see
  [Support Both Install and Import Creation Modes](creation-model.md)),
- the rootfs archive a distribution was originally imported from, or
- the install location its virtual disk lives in.

## Decision

`distribution`, `rootfs`, and `location` are all creation-time-only inputs
from Terraform's point of view: the resource schema marks them
`Optional + Computed` (required in practice to create a new resource, but
left unset rather than guessed at after `terraform import`), and `Read`
never invents values for them.

`version` and `state` are genuinely observable and are populated from
`wsl --list --verbose` on every `Read`.

Because `distribution`/`rootfs`/`location` also carry
`stringplanmodifier.RequiresReplace`-like behavior, a naive plan modifier
would propose destroying and recreating a resource the moment a
practitioner's configuration (which must supply *some* value for these
attributes to manage the resource going forward) diverges from a `null`
post-import state. `internal/provider/planmodifiers.go`
(`requiresReplaceUnlessImporting`) exists specifically to avoid that: when
the prior state value is `null` -- which only happens immediately after
`terraform import` -- it adopts the configured value into state instead of
flagging a replace. Genuine changes to an already-known value still
require replacement.

## Consequences

- Immediately after `terraform import`, `distribution`/`rootfs`/`location`
  are `null` in state. The first `terraform plan` after import needs a
  matching value written into configuration, and that first plan adopts it
  without proposing a destructive replace.
- This provider never fabricates a plausible-looking value for these
  attributes; a `null` in state honestly reflects "not knowable", rather
  than risking a wrong guess a user might trust.
- `name` is also `Optional + Computed` (an omitted `name` in install mode
  defaults to `distribution`; see
  [Support Both Install and Import Creation Modes](creation-model.md)),
  but for a different reason than the other three: it is always resolved
  to a known value by `Create` before it returns, and `ImportState`
  populates it directly from the import ID. It is never left `null` the
  way `distribution`/`rootfs`/`location` are.
