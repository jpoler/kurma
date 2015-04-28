// Copyright 2012-2014 Apcera Inc. All rights reserved.

// +build windows !cgo

package logray

import (
	"os"
)

// Since we only ever tested on Posix machines we just say false for
// windows. Once we can test on a Windows platform we can make this work.
func isTerminal(file *os.File) bool {
	return false
}
