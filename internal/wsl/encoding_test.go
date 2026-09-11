package wsl

import "testing"

func TestDecodeOutput_UTF8Passthrough(t *testing.T) {
	in := "NAME  STATE  VERSION\nUbuntu Running 2\n"
	got := decodeOutput([]byte(in))
	if got != in {
		t.Errorf("decodeOutput mangled plain UTF-8 input: got %q, want %q", got, in)
	}
}

func TestDecodeOutput_UTF16LEWithBOM(t *testing.T) {
	want := "Ubuntu\r\n"
	got := decodeOutput(utf16LEBytes(want))
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecodeOutput_UTF16LENoBOM(t *testing.T) {
	want := "NAME STATE VERSION\r\nUbuntu Running 2\r\n"
	raw := utf16LEBytes(want)[2:] // strip the BOM to test heuristic detection
	got := decodeOutput(raw)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecodeOutput_Empty(t *testing.T) {
	if got := decodeOutput(nil); got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}
