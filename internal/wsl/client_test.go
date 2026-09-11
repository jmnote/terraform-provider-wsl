package wsl

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeRunner is a scriptable Runner used to unit test client without a real
// wsl.exe or a real Windows/WSL host.
type fakeRunner struct {
	calls []([]string)
	fn    func(args []string) (Result, error)
}

func (f *fakeRunner) Run(_ context.Context, args ...string) (Result, error) {
	f.calls = append(f.calls, args)
	if f.fn != nil {
		return f.fn(args)
	}
	return Result{}, nil
}

const sampleList = "  NAME      STATE           VERSION\n" +
	"* Ubuntu    Running         2\n" +
	"  worker    Stopped         2\n"

func TestClient_List(t *testing.T) {
	runner := &fakeRunner{fn: func(args []string) (Result, error) {
		return Result{Stdout: []byte(sampleList)}, nil
	}}
	c := NewClient(runner)

	dists, err := c.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dists) != 2 {
		t.Fatalf("got %d distributions, want 2", len(dists))
	}
	if len(runner.calls) != 1 || runner.calls[0][0] != "--list" || runner.calls[0][1] != "--verbose" {
		t.Errorf("unexpected runner call: %+v", runner.calls)
	}
}

func TestClient_List_Empty(t *testing.T) {
	// wsl.exe exits non-zero, with locale-dependent text, when there are
	// zero registered distributions. List should treat that as an empty
	// list rather than surfacing an error, per the documented best-effort
	// handling in client.go.
	runner := &fakeRunner{fn: func(args []string) (Result, error) {
		return Result{
			Stdout:   []byte("Windows Subsystem for Linux has no installed distributions.\n"),
			ExitCode: 1,
		}, errors.New("exit code 1")
	}}
	c := NewClient(runner)

	dists, err := c.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dists) != 0 {
		t.Errorf("got %d distributions, want 0", len(dists))
	}
}

func TestClient_Get_Found(t *testing.T) {
	runner := &fakeRunner{fn: func(args []string) (Result, error) {
		return Result{Stdout: []byte(sampleList)}, nil
	}}
	c := NewClient(runner)

	d, err := c.Get(context.Background(), "WORKER") // exercise case-insensitivity
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Name != "worker" || d.Version != 2 {
		t.Errorf("unexpected distribution: %+v", d)
	}
}

func TestClient_Get_NotFound(t *testing.T) {
	runner := &fakeRunner{fn: func(args []string) (Result, error) {
		return Result{Stdout: []byte(sampleList)}, nil
	}}
	c := NewClient(runner)

	_, err := c.Get(context.Background(), "does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got error %v, want ErrNotFound", err)
	}
}

func TestClient_Create(t *testing.T) {
	runner := &fakeRunner{fn: func(args []string) (Result, error) {
		return Result{}, nil
	}}
	c := NewClient(runner)

	err := c.Create(context.Background(), CreateOptions{
		Name:     "my worker",
		Rootfs:   `C:\images\ubuntu 24.04.tar`,
		Location: `D:\WSL\my worker`,
		Version:  2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"--import", "my worker", `D:\WSL\my worker`, `C:\images\ubuntu 24.04.tar`, "--version", "2"}
	got := runner.calls[0]
	if len(got) != len(want) {
		t.Fatalf("args = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestClient_Create_NoVersionOmitsFlag(t *testing.T) {
	runner := &fakeRunner{fn: func(args []string) (Result, error) { return Result{}, nil }}
	c := NewClient(runner)

	if err := c.Create(context.Background(), CreateOptions{Name: "n", Rootfs: "r.tar", Location: `C:\loc`}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, a := range runner.calls[0] {
		if a == "--version" {
			t.Errorf("did not expect --version in args: %v", runner.calls[0])
		}
	}
}

func TestClient_Create_ValidatesRequiredFields(t *testing.T) {
	c := NewClient(&fakeRunner{})

	cases := []CreateOptions{
		{Rootfs: "r.tar", Location: `C:\loc`}, // missing name
		{Name: "n", Location: `C:\loc`},       // missing rootfs
		{Name: "n", Rootfs: "r.tar"},          // missing location
		{Name: "n"},                           // neither mode set
		{Name: "n", Rootfs: "r.tar", Location: `C:\loc`, Distribution: "Ubuntu"}, // both modes set
	}
	for _, opts := range cases {
		if err := c.Create(context.Background(), opts); err == nil {
			t.Errorf("Create(%+v) = nil error, want validation error", opts)
		}
	}
}

func TestClient_Create_InstallMode(t *testing.T) {
	runner := &fakeRunner{fn: func(args []string) (Result, error) { return Result{}, nil }}
	c := NewClient(runner)

	err := c.Create(context.Background(), CreateOptions{Name: "Ubuntu-24.04", Distribution: "Ubuntu-24.04"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"--install", "--distribution", "Ubuntu-24.04", "--no-launch"}
	got := runner.calls[0]
	if len(got) != len(want) {
		t.Fatalf("args = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestClient_Create_InstallMode_AppliesRequestedVersion(t *testing.T) {
	runner := &fakeRunner{fn: func(args []string) (Result, error) { return Result{}, nil }}
	c := NewClient(runner)

	err := c.Create(context.Background(), CreateOptions{Name: "Ubuntu-24.04", Distribution: "Ubuntu-24.04", Version: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runner.calls) != 2 {
		t.Fatalf("got %d runner calls, want 2 (--install then --set-version): %+v", len(runner.calls), runner.calls)
	}
	setVersion := runner.calls[1]
	want := []string{"--set-version", "Ubuntu-24.04", "1"}
	for i := range want {
		if setVersion[i] != want[i] {
			t.Errorf("set-version arg[%d] = %q, want %q", i, setVersion[i], want[i])
		}
	}
}

func TestClient_Create_InstallMode_RejectsNameMismatch(t *testing.T) {
	c := NewClient(&fakeRunner{})

	err := c.Create(context.Background(), CreateOptions{Name: "worker", Distribution: "Ubuntu-24.04"})
	if err == nil {
		t.Fatal("expected an error when name does not match distribution in install mode")
	}
	if !strings.Contains(err.Error(), "does not support installing a Store distribution under a custom name") {
		t.Errorf("error = %v, want it to explain the wsl.exe --name limitation", err)
	}
}

func TestClient_Create_Failure_PreservesStderr(t *testing.T) {
	runner := &fakeRunner{fn: func(args []string) (Result, error) {
		return Result{Stderr: []byte("distribution already exists")}, errors.New("exit code 1")
	}}
	c := NewClient(runner)

	err := c.Create(context.Background(), CreateOptions{Name: "n", Rootfs: "r.tar", Location: `C:\loc`})
	if err == nil || !strings.Contains(err.Error(), "distribution already exists") {
		t.Fatalf("err = %v, want it to contain captured stderr", err)
	}
}

func TestClient_SetVersion(t *testing.T) {
	runner := &fakeRunner{fn: func(args []string) (Result, error) { return Result{}, nil }}
	c := NewClient(runner)

	if err := c.SetVersion(context.Background(), "worker", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"--set-version", "worker", "1"}
	got := runner.calls[0]
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestClient_SetVersion_RejectsInvalidVersion(t *testing.T) {
	c := NewClient(&fakeRunner{})
	if err := c.SetVersion(context.Background(), "worker", 3); err == nil {
		t.Errorf("expected error for invalid version 3")
	}
}

func TestClient_Delete_Idempotent(t *testing.T) {
	// Distribution already absent: Delete must succeed without ever
	// calling --unregister.
	runner := &fakeRunner{fn: func(args []string) (Result, error) {
		return Result{Stdout: []byte("  NAME  STATE  VERSION\n")}, nil
	}}
	c := NewClient(runner)

	if err := c.Delete(context.Background(), "worker"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, call := range runner.calls {
		if len(call) > 0 && call[0] == "--unregister" {
			t.Errorf("did not expect --unregister to be called for an already-absent distribution")
		}
	}
}

func TestClient_Delete_Existing(t *testing.T) {
	runner := &fakeRunner{fn: func(args []string) (Result, error) {
		if len(args) > 0 && args[0] == "--list" {
			return Result{Stdout: []byte(sampleList)}, nil
		}
		return Result{}, nil
	}}
	c := NewClient(runner)

	if err := c.Delete(context.Background(), "worker"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, call := range runner.calls {
		if len(call) >= 2 && call[0] == "--unregister" && call[1] == "worker" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected --unregister worker to be called, calls: %+v", runner.calls)
	}
}
