// Copyright 2012 Apcera Inc. All rights reserved.

// +build cgo,!windows,!linux

package str

// Above may add freebsd after we test it there.
// When you change the above make sure to change term_other.go

/*
#include <unistd.h>
*/
import "C"

import (
	"os"
)

// May move to util/file? For now is here because it's the only
// one used by colors.
func IsTerminal(file *os.File) bool {
	rv, err := C.isatty(C.int(file.Fd()))
	if err != nil {
		return false
	}
	if rv != 0 {
		return true
	}
	return false
}
