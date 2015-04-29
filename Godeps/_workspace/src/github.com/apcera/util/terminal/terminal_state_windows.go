// Copyright 2014 Apcera Inc. All rights reserved.

// +build windows

// FIXME: we should implement real terminal state save/restore for Windows

package terminal

type WindowsTerminalState struct{}

func Isatty(_ uintptr) bool {
	//FIXME(Sha): Assume we're not piping output on windows.   This is possible though, and should be handled.
	return true
}

func getOSTerminalState() (WindowsTerminalState, error) {
	return WindowsTerminalState{}, nil
}

func (_ WindowsTerminalState) IsValid() bool {
	return true
}

func (_ WindowsTerminalState) Restore() {
	return
}
