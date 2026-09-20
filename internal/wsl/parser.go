package wsl

import (
	"regexp"
	"strconv"
	"strings"
)

// columnGap separates the NAME/STATE/VERSION columns of `wsl --list
// --verbose` output: wsl.exe pads each column to a fixed width, so
// consecutive columns are always separated by a run of two or more spaces,
// while a single space inside a column's own text (e.g. a distribution
// name such as "My Distro", or a two-word localized state) is preserved.
// This is what lets column splitting stay correct even when STATE is
// localized into multiple words.
var columnGap = regexp.MustCompile(`\s{2,}`)

// ParseListVerbose parses the raw output of `wsl.exe --list --verbose`
// into a list of Distributions.
//
// wsl.exe's `--list --verbose` output is a fixed-width, human-oriented
// table intended for a terminal, not a machine-readable format (there is no
// current wsl.exe option for that; see
// docs/design/decisions/locale-independent-parsing.md). Two details make
// naive parsing unsafe:
//
//  1. The header row ("NAME STATE VERSION") is localized on non-English
//     Windows installs, so it cannot be matched by literal text.
//  2. The STATE column is also localized (e.g. "Running" becomes a
//     different, sometimes multi-word, string on a Korean or Japanese
//     host), so its value can only be treated as an opaque, informational
//     string, and a naive split on any whitespace would misattribute part
//     of a multi-word state to the distribution name.
//
// This parser therefore never inspects header or state text. Columns are
// split on runs of two or more spaces (columnGap), which is the fixed-width
// padding wsl.exe uses between columns; a single embedded space (inside a
// multi-word name or a multi-word localized state) is preserved. A line is
// then only accepted as a distribution row if its last column parses as a
// bare integer -- the VERSION column, which is always ASCII digits
// regardless of Windows display language. Every other line (the header,
// blank lines, banner/error text mixed into stdout) is skipped.
func ParseListVerbose(raw []byte) ([]Distribution, error) {
	text := decodeOutput(raw)
	lines := strings.Split(text, "\n")

	var dists []Distribution
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" {
			continue
		}

		fields := columnGap.Split(line, -1)
		if len(fields) < 3 {
			continue
		}

		versionField := strings.TrimSpace(fields[len(fields)-1])
		version, err := strconv.Atoi(versionField)
		if err != nil {
			// Not a data row (most commonly the header).
			continue
		}

		state := strings.TrimSpace(fields[len(fields)-2])
		// Everything before STATE is the NAME column. It is normally a
		// single field, but is rejoined defensively in case a name itself
		// contains a run of two or more spaces.
		name := strings.Join(fields[:len(fields)-2], " ")

		isDefault := false
		if rest, ok := strings.CutPrefix(name, "*"); ok {
			isDefault = true
			name = strings.TrimSpace(rest)
		}
		if name == "" {
			continue
		}

		dists = append(dists, Distribution{
			Name:    name,
			State:   state,
			Version: version,
			Default: isDefault,
		})
	}

	return dists, nil
}

// FindByName returns the distribution in dists matching name. WSL treats
// distribution names case-insensitively, so the comparison is too.
func FindByName(dists []Distribution, name string) (Distribution, bool) {
	for _, d := range dists {
		if strings.EqualFold(d.Name, name) {
			return d, true
		}
	}
	return Distribution{}, false
}
