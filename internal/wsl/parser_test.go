package wsl

import (
	"testing"
	"unicode/utf16"
)

func utf16LEBytes(s string) []byte {
	u16 := utf16.Encode([]rune(s))
	b := make([]byte, 0, len(u16)*2+2)
	b = append(b, 0xFF, 0xFE) // BOM
	for _, u := range u16 {
		b = append(b, byte(u&0xFF), byte(u>>8))
	}
	return b
}

func TestParseListVerbose_Basic(t *testing.T) {
	raw := "  NAME      STATE           VERSION\n" +
		"* Ubuntu    Running         2\n" +
		"  Debian    Stopped         1\n"

	dists, err := ParseListVerbose([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dists) != 2 {
		t.Fatalf("got %d distributions, want 2: %+v", len(dists), dists)
	}

	if dists[0].Name != "Ubuntu" || dists[0].Version != 2 || dists[0].State != "Running" || !dists[0].Default {
		t.Errorf("unexpected first distribution: %+v", dists[0])
	}
	if dists[1].Name != "Debian" || dists[1].Version != 1 || dists[1].State != "Stopped" || dists[1].Default {
		t.Errorf("unexpected second distribution: %+v", dists[1])
	}
}

func TestParseListVerbose_NameWithSpaces(t *testing.T) {
	raw := "  NAME        STATE     VERSION\n" +
		"  My Distro   Running   2\n"

	dists, err := ParseListVerbose([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dists) != 1 {
		t.Fatalf("got %d distributions, want 1", len(dists))
	}
	if dists[0].Name != "My Distro" {
		t.Errorf("name = %q, want %q", dists[0].Name, "My Distro")
	}
}

func TestParseListVerbose_UnicodeName(t *testing.T) {
	raw := "  NAME     STATE     VERSION\n" +
		"  日本語     Running   2\n"

	dists, err := ParseListVerbose([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dists) != 1 {
		t.Fatalf("got %d distributions, want 1", len(dists))
	}
	if dists[0].Name != "日本語" {
		t.Errorf("name = %q, want %q", dists[0].Name, "日本語")
	}
}

func TestParseListVerbose_LocalizedHeaderAndState(t *testing.T) {
	// Simulates a non-English Windows locale, where both the header and
	// the (here, two-word) STATE column text are localized. Only VERSION
	// stays parseable, and column splitting relies on the multi-space
	// gutter wsl.exe pads between columns, not on single-word fields.
	raw := "  이름       상태            버전\n" +
		"* Ubuntu    실행 중         2\n"

	dists, err := ParseListVerbose([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dists) != 1 {
		t.Fatalf("got %d distributions, want 1: %+v", len(dists), dists)
	}
	if dists[0].Name != "Ubuntu" || dists[0].Version != 2 || !dists[0].Default {
		t.Errorf("unexpected distribution: %+v", dists[0])
	}
	if dists[0].State != "실행 중" {
		t.Errorf("State = %q, want %q (the full, unmangled localized value)", dists[0].State, "실행 중")
	}
}

func TestParseListVerbose_Empty(t *testing.T) {
	dists, err := ParseListVerbose([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dists) != 0 {
		t.Errorf("got %d distributions, want 0", len(dists))
	}
}

func TestParseListVerbose_NoDistributionsBanner(t *testing.T) {
	raw := "Windows Subsystem for Linux has no installed distributions.\n" +
		"Use 'wsl.exe --list --online' to list available distributions\n"

	dists, err := ParseListVerbose([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dists) != 0 {
		t.Errorf("got %d distributions, want 0: %+v", len(dists), dists)
	}
}

func TestParseListVerbose_UTF16LE(t *testing.T) {
	text := "  NAME      STATE           VERSION\r\n" +
		"* Ubuntu    Running         2\r\n"

	dists, err := ParseListVerbose(utf16LEBytes(text))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dists) != 1 {
		t.Fatalf("got %d distributions, want 1: %+v", len(dists), dists)
	}
	if dists[0].Name != "Ubuntu" || dists[0].Version != 2 {
		t.Errorf("unexpected distribution: %+v", dists[0])
	}
}

func TestFindByName_CaseInsensitive(t *testing.T) {
	dists := []Distribution{{Name: "Ubuntu-24.04"}, {Name: "worker"}}

	d, ok := FindByName(dists, "UBUNTU-24.04")
	if !ok || d.Name != "Ubuntu-24.04" {
		t.Errorf("FindByName case-insensitive lookup failed: %+v, %v", d, ok)
	}

	_, ok = FindByName(dists, "missing")
	if ok {
		t.Errorf("expected FindByName to report not-found for missing distribution")
	}
}
