# TODO

Follow-up items from code review of the WSL distribution resource
(`internal/provider/distribution_resource.go`, `internal/provider/planmodifiers.go`,
`internal/wsl/client.go`). Neither item blocks merging the current PR.

## 1. Extract a shared helper for the install-mode default name

`nameDefaultsToDistributionUnlessChangedModifier.PlanModifyString`
(`internal/provider/planmodifiers.go`) and `distribution_resource.Create`
(`internal/provider/distribution_resource.go`) both derive the default
`name` from `distribution` independently:

- Plan step: `configDistribution.ValueString()`
- Apply step: `plan.Distribution.ValueString()`

These are equivalent today, so there is no active bug. But if the
defaulting rule ever gains normalization (trimming, case-folding, a
suffix, etc.), updating only one of the two call sites would make
`terraform plan`'s displayed name diverge from what `terraform apply`
actually registers — the same class of bug fixed in this PR.

Recommended fix: introduce a `defaultNameForDistribution(distribution string) string`
helper and call it from both the plan modifier and `Create()`. Keep the
`Create()` fallback itself — `distribution` can still be Unknown at plan
time, so `Create()` needs its own resolution once `distribution` is known
at apply time.

## 2. `--list --quiet` fallback is unverified on older/localized WSL

`client.List` (`internal/wsl/client.go`) falls back to `wsl --list --quiet`
when `wsl --list --verbose` fails, treating an empty, successful quiet
output as "no distributions registered." This matches current WSL
upstream behavior:

- `--list --verbose` on an empty registry raises `WSL_E_DEFAULT_DISTRO_NOT_FOUND`.
- `--list --quiet` on an empty registry succeeds with empty output.

The fallback is fail-closed: if the quiet call also fails, or returns
non-empty output, `List` returns the original verbose error rather than
guessing the registry is empty. So there is no risk of wrongly dropping
state today.

Older WSL versions or non-English locales where `--list --quiet` might
also fail or print a localized message are not covered by this fallback,
and recovering automatically for them isn't safe to guess at — a false
positive would delete Terraform state for a distribution that still
exists. Leave this case documented rather than auto-handled; revisit only
with real Windows acceptance-test evidence of such a WSL version/locale
combination.
