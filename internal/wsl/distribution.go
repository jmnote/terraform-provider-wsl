package wsl

// Distribution represents the observable state of a WSL distribution, as
// reported by `wsl.exe --list --verbose`.
type Distribution struct {
	// Name is the distribution's registration name, e.g. "Ubuntu-24.04" or
	// "debian3". WSL treats this name case-insensitively.
	Name string

	// State is the run state column exactly as printed by wsl.exe (for
	// example "Running" or "Stopped" on an English-locale host). It is
	// opaque and MUST NOT be compared against hard-coded English strings by
	// callers: on a non-English Windows locale this text is localized, so
	// it is surfaced for information only, never used to drive control
	// flow inside this package.
	State string

	// Version is the WSL engine version the distribution runs under: 1 or
	// 2. Unlike State, this is always ASCII digits regardless of locale,
	// so it is safe to parse and compare.
	Version int

	// Default reports whether wsl.exe marked this distribution as the
	// default (the "*" column in `wsl --list --verbose`).
	Default bool
}

// CreateOptions describes a new distribution to register, using exactly one
// of two mutually exclusive creation modes -- see
// docs/design/decisions/creation-model.md for why both are supported and
// how they differ:
//
//   - Install mode (Distribution set): `wsl --install <Distribution>`, the
//     same primitive `wsl --install <Distribution>` uses day to day to
//     fetch a Microsoft Store distribution. No Rootfs/Location needed.
//     Name is optional: when empty, wsl.exe registers it under
//     Distribution itself (its own default when `--name` is omitted);
//     when set to something else, `--name` is passed to register it
//     under that instead (confirmed working against a real install).
//   - Import mode (Rootfs + Location set): `wsl --import`, bringing your
//     own root filesystem tar/tar.gz (a `.wsl` file -- a tar archive
//     Microsoft's own custom-distro tooling produces -- works here too).
//     Supports an arbitrary Name.
type CreateOptions struct {
	// Name is the distribution's registration name. Required in both
	// modes; this package itself never guesses at Terraform-facing
	// defaults, so callers must always supply it explicitly (see
	// internal/provider/instance_resource.go's Create).
	Name string

	// Distribution is a Microsoft Store distribution identifier (e.g.
	// "Ubuntu-24.04"), installed via `wsl --install <Distribution>`.
	// Install mode only; mutually exclusive with Rootfs/Location. Needs
	// no tar file and does not fetch or touch any other distribution
	// already registered on the host.
	Distribution string

	// Rootfs is the path to a tar/tar.gz root filesystem archive that
	// `wsl --import` will extract into the new distribution. Import mode
	// only; mutually exclusive with Distribution.
	Rootfs string

	// Location is the directory where WSL stores the distribution's
	// virtual disk. Import mode only; mutually exclusive with
	// Distribution. WSL creates this directory if it does not already
	// exist.
	Location string

	// Version selects WSL 1 or WSL 2 for the new distribution. Zero means
	// "let wsl.exe use its configured default version". In import mode
	// this is passed directly to `wsl --import --version`; `wsl --install`
	// has no equivalent per-distribution flag, so in install mode a
	// non-zero Version is applied with a follow-up `--set-version` call
	// after install completes.
	Version int
}
