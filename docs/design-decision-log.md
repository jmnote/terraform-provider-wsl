# Design Decision Log

This document is an index of the design decisions recorded under
[`docs/design-decisions/`](design-decisions/). These are not necessarily
fixed, and are likely to evolve and be replaced as new decisions are made,
particularly once the provider has real-world usage to inform them.

1. [Support Both Install and Import Creation Modes](design-decisions/creation-model.md) -- `wsl_distribution` supports two mutually exclusive creation modes: installing a Microsoft Store distribution with no tar file (`wsl --install`) and importing your own root filesystem tar (`wsl --import`).
2. [Use the Distribution's Registration Name as Resource Identity](design-decisions/resource-identity.md) -- the distribution's registration name is the `name` attribute and the entire classic, ID-based `terraform import` identity.
3. [Never Guess Creation-Time Attributes That WSL Cannot Report Back](design-decisions/observable-state.md) -- `distribution`, `rootfs`, and `location` are creation-time-only and never guessed at on `Read`/import; `version`/`state` are genuinely observable.
4. [Only WSL Version Supports an In-Place Update](design-decisions/update-vs-replace.md) -- every attribute requires replacing the resource except `version`, which updates in place via `wsl --set-version`.
5. [Parse `wsl --list --verbose` Without Depending on Localized Text](design-decisions/locale-independent-parsing.md) -- column splitting and a locale-invariant VERSION anchor, so parsing never depends on header or STATE text.
6. [Target Terraform Running Natively on Windows Only](design-decisions/platform-support.md) -- v0.1.0 supports `windows_amd64`/`windows_arm64` only; running Terraform inside WSL itself is out of scope.
