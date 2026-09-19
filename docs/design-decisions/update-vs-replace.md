# Only WSL Version Supports an In-Place Update

**Summary:** Every `wsl_distribution` attribute requires replacing the resource except `version`, which updates in place via `wsl --set-version`, matching what Microsoft documents as a supported (if slow) conversion between WSL 1 and 2.

---

## Background

Each `wsl_distribution` attribute needs a decision: does changing it
require destroying and recreating the resource (`RequiresReplace`), can it
be updated in place, or is it purely observational (`Computed` only)? This
has to follow real WSL semantics, not be assumed for convenience.

## Decision

| Attribute      | Behavior          | Why                                                                                                                |
| -------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------- |
| `name`         | `RequiresReplace` | WSL has no rename primitive.                                                                                      |
| `distribution` | `RequiresReplace` | Only consulted at `--install` time; changing it means installing a different Store distribution.                 |
| `rootfs`       | `RequiresReplace` | Only consulted at `--import` time; changing it means importing a different distribution.                         |
| `location`     | `RequiresReplace` | WSL has no "move" primitive; relocating means re-importing elsewhere.                                             |
| `version`      | in-place Update   | `wsl --set-version <name> <1\|2>` is Microsoft's documented, supported (if slow) conversion between WSL 1 and 2. |
| `state`        | Computed only     | Purely observational; not settable through this resource.                                                        |

`distribution`/`rootfs`/`location` use a custom plan modifier
(`requiresReplaceUnlessImporting`, see
[Never Guess Creation-Time Attributes That WSL Cannot Report Back](observable-state.md))
rather than the framework's plain `RequiresReplace`, so that adopting a
value after `terraform import` does not itself trigger a replace.

## Consequences

- `version` is the only attribute with genuine in-place update behavior;
  every other creation-time attribute change recreates the distribution.
- Switching WSL versions can be slow (Microsoft's own documentation warns
  about this for large distributions), but is still classified as Update
  rather than Replace, since it is a supported, non-destructive operation.
