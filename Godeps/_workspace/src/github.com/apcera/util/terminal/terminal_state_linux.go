// Copyright 2014-2015 Apcera Inc. All rights reserved.

// +build linux

package terminal

import (
	"syscall"
)

var (
	syscallGetTermios = syscall.TCGETS
	syscallSetTermios = syscall.TCSETS
)
