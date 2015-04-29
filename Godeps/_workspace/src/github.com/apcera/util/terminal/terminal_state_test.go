// Copyright 2014 Apcera Inc. All rights reserved.

package terminal

import (
	"os"
	"testing"
)

func TestTerminalStateCycle(t *testing.T) {
	// tests only valid when we have a controlling TTY
	// (Ideally there would be a portable isatty available to call on fd 0)
	fd, err := os.Open("/dev/tty")
	if err != nil {
		t.Skip("no controlling tty")
	}
	fd.Close()

	state, err := GetTerminalState()
	if err != nil {
		t.Fatalf("GetTerminalState errored: %s", err)
	}
	if !state.IsValid() {
		t.Fatalf("GetTerminalState result not valid %#v", state)
	}
	state.Restore()
}
