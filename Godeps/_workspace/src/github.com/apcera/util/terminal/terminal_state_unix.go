// Copyright 2014 Apcera Inc. All rights reserved.

// +build !windows

package terminal

import (
	"os"
	"syscall"
	"unsafe"
)

type UnixTerminalState struct {
	valid   bool
	ttyFile *os.File
	termios syscall.Termios
}

// isatty checks whether FD is a tty. Doesn't work on Windows.
func Isatty(fd uintptr) bool {
	var termios syscall.Termios

	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd,
		uintptr(syscallGetTermios), uintptr(unsafe.Pointer(&termios)), 0, 0, 0)

	return err == 0
}

func getOSTerminalState() (*UnixTerminalState, error) {
	uts := &UnixTerminalState{}
	fh, err := os.Open("/dev/tty")
	if err != nil {
		return uts, err
	}
	uts.ttyFile = fh

	_, _, errNum := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(uts.ttyFile.Fd()),
		uintptr(syscallGetTermios), uintptr(unsafe.Pointer(&uts.termios)), 0, 0, 0)
	// errNum is syscall.Errno which can be 0 without being an error
	if errNum != 0 {
		return uts, errNum
	}
	uts.valid = true
	return uts, nil
}

func (uts *UnixTerminalState) IsValid() bool {
	if uts == nil {
		return false
	}
	return uts.valid
}

func (uts *UnixTerminalState) Restore() {
	if !uts.IsValid() {
		return
	}
	_, _, _ = syscall.Syscall6(syscall.SYS_IOCTL, uintptr(uts.ttyFile.Fd()),
		uintptr(syscallSetTermios), uintptr(unsafe.Pointer(&uts.termios)), 0, 0, 0)
}
