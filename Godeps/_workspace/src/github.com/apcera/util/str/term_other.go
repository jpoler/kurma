// Copyright 2012 Apcera Inc. All rights reserved.

// +build windows !cgo

package str

import (
	"os"
)

// For now if not POSIX/*nix (isatty()) or we don't have cgo, we say false
// so that we don't use ANSI markup.
func IsTerminal(file *os.File) bool {
	return false
}
