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

// TestDecodeOutput_UTF16LENoBOM_CJKHeavy guards a real bug found by running
// this package's decoder against `wsl.exe --help` on a Korean-locale
// Windows host: looksLikeUTF16LE's ASCII-ratio heuristic alone
// under-detects UTF-16LE text that is mostly non-ASCII (most byte pairs
// have a non-zero high byte there), silently falling back to treating the
// raw UTF-16LE bytes as if they were already UTF-8 and producing mojibake.
// decodeOutput must also fall back to UTF-16LE whenever the raw bytes are
// not valid UTF-8, which is what actually catches this case.
func TestDecodeOutput_UTF16LENoBOM_CJKHeavy(t *testing.T) {
	want := "이 제품의 개인 정보 보호에 관한 정보는 https://aka.ms/privacy에서 확인하세요.\r\n"
	raw := utf16LEBytes(want)[2:] // strip the BOM to test heuristic detection
	got := decodeOutput(raw)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
