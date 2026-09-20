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
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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

	// Create registers a new distribution, via `wsl --install <Distribution>`
	// (Distribution set) or `wsl --import` (Rootfs + Location set); see
	// CreateOptions.
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

	// mu serializes state-mutating wsl.exe invocations and prevents them from
	// racing read snapshots. Terraform runs multiple resources' CRUD
	// concurrently (parallelism defaults to 10), and WSL's distribution
	// registration internals (the Lxss registry hive, the per-distribution
	// VHD) are not documented as safe for concurrent registration/
	// unregistration, so this provider does not assume they are. Concurrent
	// read snapshots are allowed, but reads never overlap a mutation.
	mu sync.RWMutex
}

// NewClient returns a Client that drives wsl.exe through runner.
func NewClient(runner Runner) Client {
	return &client{runner: runner}
}

func (c *client) List(ctx context.Context) ([]Distribution, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.list(ctx)
}

// list is List without locking, for callers that already hold c.mu.
func (c *client) list(ctx context.Context) ([]Distribution, error) {
	result, err := c.runner.Run(ctx, "--list", "--verbose")
	if err != nil {
		verboseErr := fmt.Errorf("wsl: list distributions: %w %s", err, describeOutput(result))

		// Some WSL versions return a non-zero status for verbose listing when
		// the registry is empty. `--list --quiet` has no such ambiguity: an
		// empty registry succeeds with empty output. Use it only as a fallback
		// after verbose failure, so normal list operations remain one process
		// invocation and service/access failures are not swallowed.
		//
		// This fallback is fail-closed: if the quiet call also fails, or
		// returns non-empty output, list returns the original verbose error
		// rather than guessing the registry is empty. Older WSL versions or
		// non-English locales where `--list --quiet` might also fail or print
		// a localized message fall into that same fail-closed path instead of
		// being auto-handled here, since a false positive would delete
		// Terraform state for a distribution that still exists. Extend this
		// only with real Windows acceptance-test evidence of such a WSL
		// version/locale combination.
		quietResult, quietErr := c.runner.Run(ctx, "--list", "--quiet")
		if quietErr == nil && strings.TrimSpace(decodeOutput(quietResult.Stdout)) == "" {
			return nil, nil
		}
		return nil, verboseErr
	}

	return ParseListVerbose(result.Stdout)
}

func (c *client) Get(ctx context.Context, name string) (*Distribution, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.get(ctx, name)
}

// get is Get without locking, for callers that already hold c.mu.
func (c *client) get(ctx context.Context, name string) (*Distribution, error) {
	dists, err := c.list(ctx)
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
	c.mu.Lock()
	defer c.mu.Unlock()

	if opts.Name == "" {
		return errors.New("wsl: create: name is required")
	}

	importMode := opts.Rootfs != "" || opts.Location != ""
	installMode := opts.Distribution != ""

	switch {
	case importMode && installMode:
		return errors.New("wsl: create: distribution and rootfs/location are mutually exclusive; set exactly one creation mode")
	case installMode:
		return c.createInstall(ctx, opts)
	case importMode:
		return c.createImport(ctx, opts)
	default:
		return errors.New("wsl: create: either distribution (install mode) or rootfs+location (import mode) is required")
	}
}

func (c *client) createInstall(ctx context.Context, opts CreateOptions) error {
	// `wsl --install` takes the distribution positionally
	// (`wsl --install <Distro> [Options...]`), not as a --distribution
	// flag -- confirmed against a real install
	// (`wsl --install ArchLinux --name <custom> --no-launch`, which
	// registers under <custom>). --name is a genuine, working --install
	// option on current wsl.exe (also confirmed against a real install);
	// earlier research suggesting it was rejected for "legacy" Store
	// distributions turned out to be outdated. See
	// docs/design/decisions/creation-model.md.
	args := []string{"--install", opts.Distribution}
	if opts.Name != opts.Distribution {
		args = append(args, "--name", opts.Name)
	}
	args = append(args, "--no-launch")

	tflog.Debug(ctx, "wsl: running wsl.exe", map[string]interface{}{"args": args})
	result, err := c.runner.Run(ctx, args...)
	if err != nil {
		return fmt.Errorf("wsl: install %q: %w %s", opts.Distribution, err, describeOutput(result))
	}

	// Unlike `wsl --import`, `wsl --install` has no per-invocation
	// --version flag: the new distribution gets whatever version
	// `wsl --set-default-version` currently points at. Apply the
	// requested version explicitly rather than leaving it to that
	// host-wide, mutable default. Call the unexported setVersion, not the
	// public SetVersion: c.mu is already held by Create, and sync.Mutex is
	// not reentrant.
	if opts.Version != 0 {
		if err := c.setVersion(ctx, opts.Name, opts.Version); err != nil {
			// The distribution now exists on the host but not at the
			// requested version, and Create is about to return an error
			// -- so the caller (internal/provider) will not save any
			// state for it. Left alone, that's an orphan: invisible to
			// Terraform, but "already exists" on the next apply's
			// `wsl --install`. Roll back by unregistering it, so Create
			// stays all-or-nothing instead of leaving a partial result
			// only discoverable via `wsl --list --verbose` by hand.
			if unregErr := c.unregister(ctx, opts.Name); unregErr != nil {
				return fmt.Errorf(
					"wsl: install %q: applying requested version: %w; additionally, rolling back the "+
						"otherwise-orphaned distribution failed: %w (you will need to run "+
						"`wsl --unregister %s` yourself)",
					opts.Distribution, err, unregErr, opts.Name,
				)
			}
			return fmt.Errorf("wsl: install %q: applying requested version: %w (rolled back: unregistered %q)",
				opts.Distribution, err, opts.Name)
		}
	}
	return nil
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

	tflog.Debug(ctx, "wsl: running wsl.exe", map[string]interface{}{"args": args})
	result, err := c.runner.Run(ctx, args...)
	if err != nil {
		return fmt.Errorf("wsl: import %q: %w %s", opts.Name, err, describeOutput(result))
	}
	return nil
}

func (c *client) SetVersion(ctx context.Context, name string, version int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.setVersion(ctx, name, version)
}

// setVersion is SetVersion without locking c.mu, for use by callers (like
// createInstall) that already hold it.
func (c *client) setVersion(ctx context.Context, name string, version int) error {
	if version != 1 && version != 2 {
		return fmt.Errorf("wsl: set-version: version must be 1 or 2, got %d", version)
	}

	args := []string{"--set-version", name, strconv.Itoa(version)}
	tflog.Debug(ctx, "wsl: running wsl.exe", map[string]interface{}{"args": args})
	result, err := c.runner.Run(ctx, args...)
	if err != nil {
		return fmt.Errorf("wsl: set-version %q to %d: %w %s", name, version, err, describeOutput(result))
	}
	return nil
}

func (c *client) Delete(ctx context.Context, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check existence ourselves first, rather than inferring success from
	// wsl.exe's (locale-dependent) message text, so Delete stays
	// idempotent without any text matching.
	_, err := c.get(ctx, name)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	return c.unregister(ctx, name)
}

// unregister runs `wsl --unregister name` without locking c.mu, for use by
// callers (Delete, createInstall's rollback) that already hold it.
func (c *client) unregister(ctx context.Context, name string) error {
	args := []string{"--unregister", name}
	tflog.Debug(ctx, "wsl: running wsl.exe", map[string]interface{}{"args": args})
	result, err := c.runner.Run(ctx, args...)
	if err != nil {
		return fmt.Errorf("wsl: unregister %q: %w %s", name, err, describeOutput(result))
	}
	return nil
}

// describeOutput formats a command's captured output for an error message.
// wsl.exe is a Windows console application and does not reliably follow
// the Unix convention of writing error text to stderr specifically --
// several real failures (a duplicate distribution name, a bad argument)
// have been observed to print to stdout instead -- so error messages
// include whichever of stdout/stderr is non-empty rather than only stderr.
func describeOutput(result Result) string {
	stderr := decodeOutput(result.Stderr)
	stdout := decodeOutput(result.Stdout)
	switch {
	case stderr != "" && stdout != "":
		return fmt.Sprintf("(stderr: %s, stdout: %s)", stderr, stdout)
	case stderr != "":
		return fmt.Sprintf("(stderr: %s)", stderr)
	case stdout != "":
		return fmt.Sprintf("(stdout: %s)", stdout)
	default:
		return "(no output captured)"
	}
}
