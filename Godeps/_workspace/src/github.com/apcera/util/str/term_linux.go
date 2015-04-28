// Copyright 2012 Apcera Inc. All rights reserved.

// +build linux

package str

// BSD implements isatty() by testing if tcgetattr() errors; some experimental
// Go code from Google (Ian Lance Taylor) used to do the same thing.

// FreeBSD implements tcgetattr() with TIOCGETA ioctl, but on Ubuntu Linux,
// tty_ioctl(4) documents this as being the TCGETS ioctl instead.  For now,
// we only care about Linux.

import (
	"os"
	"syscall"
	"unsafe"
)

// May move to util/file? For now is here because it's the only
// one used by colors.
func IsTerminal(file *os.File) bool {
	var tios syscall.Termios
	_, _, ep := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(file.Fd()), syscall.TCGETS, uintptr(unsafe.Pointer(&tios)))
	if ep != 0 {
		// syscall failed, it's not a terminal
		return false
	}
	return true
}
