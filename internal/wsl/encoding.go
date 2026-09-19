package wsl

import (
	"unicode/utf16"
	"unicode/utf8"
)

// decodeOutput converts raw bytes captured from wsl.exe's stdout/stderr into
// a Go string.
//
// wsl.exe is inconsistent about the encoding it writes: when attached to a
// real console it writes the active console code page (commonly UTF-8 on
// modern Windows), but when its stdout/stderr are redirected to a pipe (as
// os/exec always does) it frequently writes UTF-16LE instead, with or
// without a byte-order mark. Decoding UTF-8 bytes as UTF-16LE (or vice
// versa) produces mojibake or a garbled distribution list, which in turn
// breaks state reconciliation, so this detection has to run before any
// parsing.
//
// Detecting the no-BOM case needs two complementary signals, not one:
//
//   - looksLikeUTF16LE catches the common case this package actually
//     parses -- ASCII-heavy text (distribution names, "Running"/"Stopped",
//     digits) encoded as UTF-16LE, which shows a very characteristic
//     pattern of alternating printable bytes and 0x00 bytes.
//   - validUTF8AllowingTruncatedTail(b) being false catches the opposite
//     case: UTF-16LE text that is mostly non-ASCII (e.g. wsl.exe's --help
//     output on a non-English Windows display language, verified against a
//     real Korean-locale host during development). There, most byte pairs
//     have a non-zero high byte, so looksLikeUTF16LE's ratio check alone
//     under-detects it -- but that same byte pattern also happens to
//     violate UTF-8's encoding rules, which validUTF8AllowingTruncatedTail
//     catches instead (it is plain utf8.Valid, except that it does not
//     mistake a context-deadline-truncated UTF-8 sequence at the very end
//     of b for genuinely non-UTF-8 bytes).
//
// Neither signal alone is reliable across both shapes of content (an
// ASCII-heavy UTF-16LE stream trivially satisfies utf8.Valid, since every
// individual byte -- printable ASCII or 0x00 -- is independently a valid
// one-byte UTF-8 sequence), so both are checked.
func decodeOutput(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	// Explicit UTF-16LE byte-order mark.
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
		return decodeUTF16LE(b[2:])
	}

	if len(b) >= 2 && (looksLikeUTF16LE(b) || !validUTF8AllowingTruncatedTail(b)) {
		return decodeUTF16LE(b)
	}

	return string(b)
}

// validUTF8AllowingTruncatedTail reports whether b is valid UTF-8, treating
// a trailing incomplete multi-byte sequence as valid rather than corrupt.
//
// A context deadline or process kill can cut off the captured output while
// wsl.exe is mid-write of a multi-byte UTF-8 character, leaving b ending in
// a truncated (but otherwise well-formed) sequence. Plain utf8.Valid(b)
// treats that the same as genuinely non-UTF-8 bytes (e.g. real UTF-16LE
// output), which would send an otherwise-readable, merely-truncated error
// message through decodeUTF16LE and turn it into mojibake -- precisely in
// the timeout/cancellation path where a clear message matters most.
func validUTF8AllowingTruncatedTail(b []byte) bool {
	if utf8.Valid(b) {
		return true
	}

	// The only incomplete sequence a truncation can leave is at the very
	// end, and is at most 3 bytes (the longest UTF-8 sequence is 4 bytes).
	maxTail := 3
	if maxTail > len(b) {
		maxTail = len(b)
	}
	for cut := 1; cut <= maxTail; cut++ {
		head, tail := b[:len(b)-cut], b[len(b)-cut:]
		if utf8.Valid(head) && isIncompleteUTF8Sequence(tail) {
			return true
		}
	}
	return false
}

// isIncompleteUTF8Sequence reports whether tail is a well-formed prefix of
// a multi-byte UTF-8 sequence that is missing one or more trailing
// continuation bytes -- as opposed to bytes that are not part of any valid
// UTF-8 sequence regardless of what might follow them.
func isIncompleteUTF8Sequence(tail []byte) bool {
	if len(tail) == 0 {
		return false
	}

	var wantLen int
	switch lead := tail[0]; {
	case lead&0xE0 == 0xC0:
		wantLen = 2
	case lead&0xF0 == 0xE0:
		wantLen = 3
	case lead&0xF8 == 0xF0:
		wantLen = 4
	default:
		return false
	}
	if wantLen <= len(tail) {
		return false // a complete sequence would already have been valid
	}
	for _, c := range tail[1:] {
		if c&0xC0 != 0x80 {
			return false
		}
	}
	return true
}

func looksLikeUTF16LE(b []byte) bool {
	n := len(b)
	if n < 4 {
		return false
	}
	// Inspect a bounded prefix; the heuristic only needs to be right about
	// the general shape of the stream, not every byte.
	sample := n
	if sample > 256 {
		sample = 256
	}
	sample -= sample % 2

	zeroOddCount := 0
	pairs := 0
	for i := 0; i+1 < sample; i += 2 {
		pairs++
		lo, hi := b[i], b[i+1]
		if hi == 0x00 && (lo == 0x00 || lo >= 0x09) {
			zeroOddCount++
		}
	}
	if pairs == 0 {
		return false
	}
	// Require an overwhelming majority to avoid misdetecting normal UTF-8
	// text that merely happens to contain a stray null.
	return float64(zeroOddCount)/float64(pairs) > 0.9
}

func decodeUTF16LE(b []byte) string {
	if len(b)%2 != 0 {
		b = b[:len(b)-1]
	}
	u16 := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u16 = append(u16, uint16(b[i])|uint16(b[i+1])<<8)
	}
	return string(utf16.Decode(u16))
}
