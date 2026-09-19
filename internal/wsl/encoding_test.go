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

// TestDecodeOutput_TruncatedUTF8NotMisreadAsUTF16LE guards against a real
// bug: a context deadline or process kill can cut off captured output
// mid-write of a multi-byte UTF-8 character, leaving the buffer ending in
// an incomplete (but not corrupt) UTF-8 sequence. Plain utf8.Valid treats
// that the same as genuinely non-UTF-8 bytes, which used to send the
// otherwise-readable, merely-truncated text through decodeUTF16LE and turn
// it into mojibake -- precisely in the timeout/cancellation path where a
// clear message matters most.
func TestDecodeOutput_TruncatedUTF8NotMisreadAsUTF16LE(t *testing.T) {
	full := []byte("distribution name: 한")
	// Cut off the last byte of the trailing multi-byte character so the
	// buffer ends in an incomplete (not corrupt) UTF-8 sequence.
	truncated := full[:len(full)-1]

	got := decodeOutput(truncated)
	want := string(truncated)
	if got != want {
		t.Errorf("decodeOutput misread truncated UTF-8 as UTF-16LE: got %q, want %q", got, want)
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
