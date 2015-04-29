// Copyright 2014 Apcera Inc. All rights reserved.

package terminal

import "os"

// TerminalState can be used to restore terminal state.
type TerminalState interface {
	IsValid() bool
	Restore()
}

func GetTerminalState() (TerminalState, error) {
	return getOSTerminalState()
}

var stdoutIsTTY = Isatty(os.Stdout.Fd())
