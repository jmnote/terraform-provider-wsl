# Design Decisions

| Decision | Summary |
| --- | --- |
| [Support Both Install and Import Creation Modes](creation-model.md) | `wsl_instance` supports two mutually exclusive creation modes: installing a Microsoft Store distribution with no tar file (`wsl --install`) and importing your own root filesystem tar (`wsl --import`). |
| [Use the Instance's Registration Name as Resource Identity](resource-identity.md) | The instance's registration name is the `name` attribute and the entire classic, ID-based `terraform import` identity. |
| [Never Guess Creation-Time Attributes That WSL Cannot Report Back](observable-state.md) | `distribution`, `rootfs`, and `location` are creation-time-only and never guessed at on `Read`/import; `version`/`state` are genuinely observable. |
| [Only WSL Version Supports an In-Place Update](update-vs-replace.md) | Every attribute requires replacing the resource except `version`, which updates in place via `wsl --set-version`. |
| [Parse `wsl --list --verbose` Without Depending on Localized Text](locale-independent-parsing.md) | Column splitting and a locale-invariant VERSION anchor, so parsing never depends on header or STATE text. |
| [Target Terraform Running Natively on Windows Only](platform-support.md) | v0.1.0 supports `windows_amd64`/`windows_arm64` only; running Terraform inside WSL itself is out of scope. |
| [Require an Explicit `name` in Every Creation Mode](required-name.md) | `name` is `Required` in both modes, with no default from `distribution`. |
| [Call the Resource `wsl_instance`, Not `wsl_distribution`](instance-terminology.md) | The resource/data source are named `wsl_instance` to avoid colliding with the `distribution` attribute's own meaning. |
