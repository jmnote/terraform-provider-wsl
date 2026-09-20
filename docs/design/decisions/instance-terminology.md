# Call the Resource `wsl_instance`, Not `wsl_distribution`

**Summary:** The resource and data source are named `wsl_instance`, even
though WSL's own documentation and `wsl.exe` call the same thing a
"distribution" or "distro."

---

## Background

WSL's own vocabulary uses "distribution" for two different things: the
Store catalog entry an instance is installed *from* (`wsl --install
<Distribution>`), and the registered, named, stateful thing that
installation produces (what `wsl --list --verbose` lists, what `wsl
--unregister` deletes). This provider's schema needs to keep them apart:
it has its own `distribution` attribute (the catalog identifier) inside
the resource that manages the registered thing. If the resource itself
were also `wsl_distribution`, `wsl_distribution.ubuntu.distribution` would
use the same word for the resource type and one of its own attributes, for
two different concepts.

The closer analogy is a cloud provider's image/instance split: an image is
a template; an instance is the thing you actually launch, name, and tear
down, potentially several from the same image.

## Decision

The resource and data source are `wsl_instance`. `distribution` keeps its
name as the attribute identifying which Store image to install from
(install mode only; `rootfs`/`location` are the import-mode equivalent).
`internal/wsl` (the `wsl.exe` wrapper layer) keeps its own `Distribution`
type and `CreateOptions.Distribution` field unchanged, since it mirrors
`wsl.exe`'s own vocabulary one layer below this naming choice.
