# Proposal: Image Cache Policy and Digest Pinning (not accepted)

**Status:** Proposed, not implemented. This is a discussion record, not a
design decision -- unlike `docs/design-decisions/`, nothing here is
shipped in v0.1.0. Kept separate from that directory for exactly that
reason: it should not read as settled.

## Original idea

Model a WSL distribution's source image the way container registries
model images: an acquisition policy independent of which distribution is
requested, plus optional content pinning.

```hcl
resource "wsl_instance" "example" {
  distribution = "ArchLinux"
  cache_policy = "prefer" # never | prefer | refresh
  digest       = "sha256:..."
}
```

| `cache_policy` | Behavior                                                                |
| -------------- | ------------------------------------------------------------------------ |
| `never`        | Always re-download the image, ignoring any local cache.                |
| `prefer`       | Use a local cache if present; otherwise download and cache it.         |
| `refresh`      | Check the remote catalog for a newer image; download if the cache is stale. |

`digest`, independent of `cache_policy`, would pin to a specific image
content hash and verify whatever is cached/downloaded against it.
Computed attributes (`image_url`, `image_digest`, `image_cached`) would
report what was actually used. Three orthogonal axes: `distribution`
(which distribution), `cache_policy` (how to acquire its image),
`digest` (exactly which image).

## Why this doesn't fit v0.1.0 as designed

`wsl.exe` has no local image cache concept separate from a registered
distribution *instance* the way Docker separates a pulled image from a
running container: `wsl --install` downloads and registers in one
operation, and a second `--install` of the same distribution creates a
second, independently-downloaded registration, not a second instance
sharing one cached image. There is also no `--digest`-style flag for
integrity pinning from outside; `DistributionInfo.json`'s own `Sha256`
values are Microsoft's internal integrity check on the download, not a
verification knob this provider (or a user) can drive.

Implementing `cache_policy`/`digest` for real would mean this provider
takes on the acquisition step itself, rather than delegating it to
`wsl.exe`: parsing
[`DistributionInfo.json`](https://github.com/microsoft/WSL/blob/master/distributions/DistributionInfo.json)
to resolve a flavor to a concrete URL/hash, an HTTP client with checksum
verification, and a local cache directory/eviction scheme -- only the
final registration step (`wsl --install --from-file <cached-path>`, or
`wsl --import`) would still go through `wsl.exe`. That is not "one more
option on install mode"; it moves the provider's responsibility boundary
from *a thin, declarative wrapper around wsl.exe* to *an image
downloader/cache with a thin wrapper on top*, which is a materially
different (and materially larger) thing to build and maintain than what
`docs/design-decisions/` describes as this provider's scope.

## What actually shipped instead

A simplified version of the same instinct -- "let a user manage their own
image instead of relying on `distribution`" -- without any new schema
surface: `rootfs` (import mode) already accepts a `.wsl` file directly.
`.wsl` is just a tar archive with a renamed extension (Microsoft's own
custom-distro packaging format), and this was verified against a real
file: a genuine ArchLinux `.wsl` download was fed straight to
`wsl --import <Name> <Location> <file>.wsl` and registered correctly with
no errors. So a user who wants full control over exactly which image
bytes get used already has it, today, via `rootfs` -- no `image` attribute
needed. See `docs/design-decisions/creation-model.md` and the resource
docs' "Import mode" section.

## Open questions, if this is revisited

- Is provider-managed caching/pinning worth the scope expansion above, or
  does "point `rootfs` at whatever local file you already trust" cover
  the real need well enough in practice?
- If pursued: does it become a third creation mode (`image` +
  `cache_policy`), or a variant of import mode specifically (since it
  would still end in `wsl --import`/`--install --from-file` either way)?
- Where would the cache directory live, and what's the eviction/sizing
  story for potentially multi-hundred-MB rootfs images?
- Would digest pinning check the *catalog's* published hash (trusting
  Microsoft's manifest), an independently-computed hash of the downloaded
  bytes, or both?
- Is there real, observed pain from repeated re-downloads (the practical
  problem `cache_policy` would solve) once this provider has real-world
  usage, or is this solving a problem nobody has hit yet?

Revisit only if a real, observed need shows up -- not preemptively.
