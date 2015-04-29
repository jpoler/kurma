// Copyright 2014-2015 Apcera Inc. All rights reserved.

// +build darwin

package terminal

import (
	"syscall"
)

var (
	syscallGetTermios uint64 = syscall.TIOCGETA
	syscallSetTermios uint64 = syscall.TIOCSETA
)
