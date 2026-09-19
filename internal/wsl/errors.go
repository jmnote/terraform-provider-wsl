package wsl

import "errors"

// ErrNotFound is returned by Client.Get when no distribution with the
// requested name is currently registered with WSL.
var ErrNotFound = errors.New("wsl: distribution not found")
