package wsl

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// These tests exercise the real ProcessRunner (no wsl.exe involved) using
// cmd.exe, which every target platform (windows_amd64, windows_arm64)
// ships. This validates process execution -- stdout/stderr capture, exit
// codes, a missing executable, and context cancellation -- independently of
// WSL being installed on the machine running `go test`.

func TestProcessRunner_Stdout(t *testing.T) {
	r := NewProcessRunner("cmd.exe")
	result, err := r.Run(context.Background(), "/c", "echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(result.Stdout), "hello") {
		t.Errorf("stdout = %q, want it to contain %q", result.Stdout, "hello")
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
}

func TestProcessRunner_ExitCodeAndStderr(t *testing.T) {
	r := NewProcessRunner("cmd.exe")
	result, err := r.Run(context.Background(), "/c", "echo failure 1>&2 & exit 3")
	if err == nil {
		t.Fatal("expected an error for a non-zero exit code")
	}
	if result.ExitCode != 3 {
		t.Errorf("exit code = %d, want 3", result.ExitCode)
	}
	if !strings.Contains(string(result.Stderr), "failure") {
		t.Errorf("stderr = %q, want it to contain %q", result.Stderr, "failure")
	}
}

func TestProcessRunner_ExecutableNotFound(t *testing.T) {
	r := NewProcessRunner("definitely-not-a-real-executable-xyz.exe")
	_, err := r.Run(context.Background(), "--version")
	if !errors.Is(err, ErrExecutableNotFound) {
		t.Fatalf("err = %v, want it to wrap ErrExecutableNotFound", err)
	}
}

func TestLastVisibleLine(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"single line", "hello", "hello"},
		{"trailing newline", "hello\n", "hello"},
		{
			name: "progress bar rewritten with bare CR",
			in:   "Downloading: Debian GNU/Linux\r[==========50.0%          ]\r[==================100.0%]",
			want: "[==================100.0%]",
		},
		{
			name: "CRLF terminated lines",
			in:   "step one\r\nstep two\r\n",
			want: "step two",
		},
		{"blank after last content", "done\r\n\r\n", "done"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lastVisibleLine(tt.in); got != tt.want {
				t.Errorf("lastVisibleLine(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestProcessRunner_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// ping.exe is invoked directly, not through cmd.exe: killing a cmd.exe
	// wrapper leaves ping.exe running as an orphaned grandchild that keeps
	// the captured stdout/stderr pipes open until it finishes on its own,
	// which would make this test hang for the ping's full duration instead
	// of returning promptly on cancellation.
	r := NewProcessRunner("ping.exe")
	done := make(chan error, 1)
	go func() {
		// A ping that would otherwise run for ~30 seconds; canceling the
		// context should kill it well before that.
		_, err := r.Run(ctx, "-n", "30", "127.0.0.1")
		done <- err
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error after context cancellation")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return promptly after context cancellation")
	}
}
