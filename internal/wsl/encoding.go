package wsl

import "unicode/utf16"

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
func decodeOutput(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	// Explicit UTF-16LE byte-order mark.
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
		return decodeUTF16LE(b[2:])
	}

	// No BOM: heuristically detect UTF-16LE. wsl.exe's output is
	// overwhelmingly ASCII (distribution names, "Running"/"Stopped",
	// digits), so plain ASCII encoded as UTF-16LE shows a very
	// characteristic pattern: every even-indexed byte is a printable
	// ASCII/whitespace byte and every odd-indexed byte is 0x00.
	if len(b) >= 2 && looksLikeUTF16LE(b) {
		return decodeUTF16LE(b)
	}

	return string(b)
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
