// Copyright 2012-2014 Apcera Inc. All rights reserved.

// +build cgo,!windows

package logray

/*
#include <unistd.h>
*/
import "C"

import (
	"os"
)

// Returns true if the given file object is a Terminal.
func isTerminal(file *os.File) bool {
	rv, err := C.isatty(C.int(file.Fd()))
	if err != nil {
		return false
	}
	if rv != 0 {
		return true
	}
	return false
}
