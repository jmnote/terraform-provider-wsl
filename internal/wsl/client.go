// Package wsl isolates every interaction with wsl.exe behind a small,
// Terraform-agnostic interface (Client). internal/provider depends on this
// package; this package must never import anything from
// terraform-plugin-framework, so that List/Get/Create/SetVersion/Delete and
// the output parsing they rely on can be unit tested without Terraform, and
// without a real Windows/WSL host, by substituting a fake Runner.
package wsl

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

// Client is the abstraction internal/provider programs against. The only
// production implementation (client, via NewClient) calls wsl.exe through a
// Runner; tests substitute a fake Runner instead of a fake Client, so the
// argument-building and output-parsing logic under test is the same code
// path the provider actually runs.
type Client interface {
	// List returns every WSL distribution currently registered on the
	// host.
	List(ctx context.Context) ([]Distribution, error)

	// Get returns the distribution named name. It returns ErrNotFound
	// (checkable with errors.Is) if no such distribution is registered.
	Get(ctx context.Context, name string) (*Distribution, error)

	// Create registers a new distribution, via `wsl --import` (Rootfs +
	// Location set) or `wsl --install --distribution` (Distribution set);
	// see CreateOptions.
	Create(ctx context.Context, opts CreateOptions) error

	// SetVersion switches an existing distribution between WSL 1 and
	// WSL 2 via `wsl --set-version`.
	SetVersion(ctx context.Context, name string, version int) error

	// Delete unregisters a distribution via `wsl --unregister`. Delete is
	// idempotent: deleting a distribution that is already absent returns
	// nil rather than an error.
	Delete(ctx context.Context, name string) error
}

type client struct {
	runner Runner
}

// NewClient returns a Client that drives wsl.exe through runner.
func NewClient(runner Runner) Client {
	return &client{runner: runner}
}

func (c *client) List(ctx context.Context) ([]Distribution, error) {
	result, err := c.runner.Run(ctx, "--list", "--verbose")
	if err != nil {
		// wsl.exe exits non-zero, with a localized message, when zero
		// distributions are registered at all -- there is currently no
		// machine-readable way to distinguish that from a real failure
		// (see https://github.com/microsoft/WSL/issues/6235, and
		// docs/design-decisions/locale-independent-parsing.md). Best
		// effort: if stdout still parses into zero rows, treat it as an
		// empty list; otherwise this is a genuine failure (wsl.exe
		// missing, WSL not installed, etc.) and is surfaced to the caller
		// with the captured output attached.
		dists, parseErr := ParseListVerbose(result.Stdout)
		if parseErr == nil && len(dists) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("wsl: list distributions: %w (stderr: %s)", err, decodeOutput(result.Stderr))
	}

	return ParseListVerbose(result.Stdout)
}

func (c *client) Get(ctx context.Context, name string) (*Distribution, error) {
	dists, err := c.List(ctx)
	if err != nil {
		return nil, err
	}

	d, ok := FindByName(dists, name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return &d, nil
}

func (c *client) Create(ctx context.Context, opts CreateOptions) error {
	if opts.Name == "" {
		return errors.New("wsl: create: name is required")
	}

	importMode := opts.Rootfs != "" || opts.Location != ""
	installMode := opts.Distribution != ""

	switch {
	case importMode && installMode:
		return errors.New("wsl: create: rootfs/location and distribution are mutually exclusive; set exactly one creation mode")
	case importMode:
		return c.createImport(ctx, opts)
	case installMode:
		return c.createInstall(ctx, opts)
	default:
		return errors.New("wsl: create: either rootfs+location (import mode) or distribution (install mode) is required")
	}
}

func (c *client) createImport(ctx context.Context, opts CreateOptions) error {
	if opts.Rootfs == "" {
		return errors.New("wsl: create: rootfs is required for import mode")
	}
	if opts.Location == "" {
		return errors.New("wsl: create: location is required for import mode")
	}

	args := []string{"--import", opts.Name, opts.Location, opts.Rootfs}
	if opts.Version != 0 {
		args = append(args, "--version", strconv.Itoa(opts.Version))
	}

	result, err := c.runner.Run(ctx, args...)
	if err != nil {
		return fmt.Errorf("wsl: import %q: %w (stderr: %s)", opts.Name, err, decodeOutput(result.Stderr))
	}
	return nil
}

func (c *client) createInstall(ctx context.Context, opts CreateOptions) error {
	// wsl.exe does not support --name when installing a legacy Store
	// distribution (confirmed during research; see
	// docs/design-decisions/creation-model.md), so the registration name
	// is always exactly the Store distribution identifier. Enforcing this
	// here, with a clear error, is better than silently registering the
	// distribution under a different name than the resource's `name`
	// attribute claims.
	if opts.Name != opts.Distribution {
		return fmt.Errorf(
			"wsl: create: install mode registers the distribution under its Store identifier %q, "+
				"which does not match name %q; wsl.exe does not support installing a Store distribution "+
				"under a custom name, so name must equal distribution in install mode",
			opts.Distribution, opts.Name,
		)
	}

	result, err := c.runner.Run(ctx, "--install", "--distribution", opts.Distribution, "--no-launch")
	if err != nil {
		return fmt.Errorf("wsl: install %q: %w (stderr: %s)", opts.Distribution, err, decodeOutput(result.Stderr))
	}

	// Unlike `wsl --import`, `wsl --install` has no per-invocation
	// --version flag: the new distribution gets whatever version
	// `wsl --set-default-version` currently points at. Apply the
	// requested version explicitly rather than leaving it to that
	// host-wide, mutable default.
	if opts.Version != 0 {
		if err := c.SetVersion(ctx, opts.Name, opts.Version); err != nil {
			return fmt.Errorf("wsl: install %q: applying requested version: %w", opts.Distribution, err)
		}
	}
	return nil
}

func (c *client) SetVersion(ctx context.Context, name string, version int) error {
	if version != 1 && version != 2 {
		return fmt.Errorf("wsl: set-version: version must be 1 or 2, got %d", version)
	}

	result, err := c.runner.Run(ctx, "--set-version", name, strconv.Itoa(version))
	if err != nil {
		return fmt.Errorf("wsl: set-version %q to %d: %w (stderr: %s)", name, version, err, decodeOutput(result.Stderr))
	}
	return nil
}

func (c *client) Delete(ctx context.Context, name string) error {
	// Check existence ourselves first, rather than inferring success from
	// wsl.exe's (locale-dependent) message text, so Delete stays
	// idempotent without any text matching.
	_, err := c.Get(ctx, name)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	result, err := c.runner.Run(ctx, "--unregister", name)
	if err != nil {
		return fmt.Errorf("wsl: unregister %q: %w (stderr: %s)", name, err, decodeOutput(result.Stderr))
	}
	return nil
}
