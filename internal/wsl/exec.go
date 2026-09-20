package wsl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Result is the outcome of running a wsl.exe invocation.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Runner executes wsl.exe (or a stand-in for it in tests) and returns its
// captured output. Implementations must not use a shell: args are passed
// through to the child process argv directly, so callers never need to
// quote or escape names or Windows paths that contain spaces.
type Runner interface {
	Run(ctx context.Context, args ...string) (Result, error)
}

// ProcessRunner is the production Runner: it invokes a real executable
// (normally "wsl.exe") as a child process. It never shells out through
// cmd.exe or PowerShell, so argument quoting is handled entirely by
// os/exec's argv-based process creation.
type ProcessRunner struct {
	// Executable is the program to run, either a bare name resolved
	// against PATH (e.g. "wsl.exe") or an absolute path.
	Executable string
}

// NewProcessRunner returns a ProcessRunner that invokes executable. If
// executable is empty, "wsl.exe" is used.
func NewProcessRunner(executable string) *ProcessRunner {
	if executable == "" {
		executable = "wsl.exe"
	}
	return &ProcessRunner{Executable: executable}
}

// ErrExecutableNotFound wraps errors from a missing wsl.exe so callers can
// give a diagnostic pointing at WSL setup rather than a raw OS error.
var ErrExecutableNotFound = errors.New("wsl: executable not found")

func (r *ProcessRunner) Run(ctx context.Context, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, r.Executable, args...)

	var stdout, stderr bytes.Buffer

	// wsl.exe reports long-running operations (e.g. `--install`'s download
	// and install phases) as a sequence of lines it rewrites in place with
	// carriage returns, like a progress bar -- normally only visible on a
	// real console, and otherwise invisible since Result is only inspected
	// by the caller after the whole command finishes. progressLogger mirrors
	// each write into tflog as it arrives, so `TF_LOG=DEBUG` surfaces that
	// same progress instead of a multi-minute silence.
	var progressMu sync.Mutex
	lastLogged := ""
	logProgress := func(buf *bytes.Buffer) {
		progressMu.Lock()
		defer progressMu.Unlock()
		// decodeOutput is safe to call on a still-growing (and therefore
		// possibly mid-character) buffer: it already has to tolerate a
		// truncated tail for the context-cancellation case below.
		line := lastVisibleLine(decodeOutput(buf.Bytes()))
		if line == "" || line == lastLogged {
			return
		}
		lastLogged = line
		tflog.Debug(ctx, "wsl: progress", map[string]interface{}{"line": line})
	}
	cmd.Stdout = &progressWriter{buf: &stdout, onWrite: func() { logProgress(&stdout) }}
	cmd.Stderr = &progressWriter{buf: &stderr, onWrite: func() { logProgress(&stderr) }}

	err := cmd.Run()

	result := Result{
		Stdout: stdout.Bytes(),
		Stderr: stderr.Bytes(),
	}

	if err == nil {
		result.ExitCode = 0
		return result, nil
	}

	var notFound *exec.Error
	if errors.As(err, &notFound) {
		return result, fmt.Errorf("%w: %s: %w", ErrExecutableNotFound, r.Executable, notFound.Err)
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		// A non-zero exit is a normal, expected outcome for several WSL
		// commands (e.g. querying a distribution that does not exist), so
		// it is returned as an error the caller can inspect alongside the
		// captured stdout/stderr rather than only a bare Go error value.
		return result, fmt.Errorf("wsl: %s %v: exit code %d: %w", r.Executable, args, result.ExitCode, err)
	}

	// context.Canceled / context.DeadlineExceeded surface here.
	return result, fmt.Errorf("wsl: %s %v: %w", r.Executable, args, err)
}

// progressWriter mirrors every write into buf (preserving the existing
// full-capture behavior Result relies on) and additionally invokes onWrite
// after each one, so a caller can observe the output as it streams in
// rather than only once the command exits.
type progressWriter struct {
	buf     *bytes.Buffer
	onWrite func()
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n, err := w.buf.Write(p)
	if w.onWrite != nil {
		w.onWrite()
	}
	return n, err
}

// lastVisibleLine returns the last non-blank line in s, treating both "\n"
// and a bare "\r" as line separators. wsl.exe rewrites progress-bar lines in
// place with a bare "\r" (no "\n"), so splitting on "\n" alone would see the
// whole run as a single line and never report an updated percentage.
func lastVisibleLine(s string) string {
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return ""
}
