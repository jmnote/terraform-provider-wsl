# Parse `wsl --list --verbose` Without Depending on Localized Text

**Summary:** `internal/wsl/parser.go` splits columns on runs of two or more spaces and anchors on the always-ASCII VERSION column, so it never depends on the header text or the STATE column's values, both of which are localized on non-English Windows.

---

## Background

`wsl --list --verbose` is a fixed-width table meant for a terminal, not a
machine-readable format -- there is currently no `--json` (or similar)
output mode; see [microsoft/WSL#6235](https://github.com/microsoft/WSL/issues/6235),
which is still open. Two things about that table are localized on a
non-English Windows install: the header text and the STATE column's
values (e.g. "Running" is a different, sometimes multi-word, string in
other languages).

## Decision

`internal/wsl/parser.go` is written to never depend on header or state
text:

- Every candidate line is split on runs of two or more spaces -- the
  fixed-width padding `wsl.exe` uses between columns -- which correctly
  keeps a multi-word name or a multi-word localized state together as one
  field.
- A line is only accepted as a distribution row if its last field parses
  as a bare integer: the VERSION column is always ASCII digits regardless
  of display language, so this reliably distinguishes data rows from the
  (possibly localized) header, blank lines, and banner text, without ever
  reading header or state text.
- The parsed `state` value is carried through as an opaque, informational
  string. Nothing in this provider compares it against English words like
  "Running"; existence and identity checks use the presence of a row for
  the requested name, never state text.

