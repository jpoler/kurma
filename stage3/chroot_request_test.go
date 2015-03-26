// Copyright 2013-2015 Apcera Inc. All rights reserved.

// +build linux,cgo

package stage3

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/apcera/util/uuid"

	. "github.com/apcera/util/testtool"
)

func TestChrootRequest(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	t.Skip("FIXME: investigate issue with unit tests erroring on pivot_root with \"Device or resource busy\"")

	// Start the initd process.
	_, socket, _, pid := StartInitd(t)

	chrootDir, err := ioutil.TempDir("/var", "container"+uuid.Variant4().String())
	TestExpectSuccess(t, err)
	AddTestFinalizer(func() { os.RemoveAll(chrootDir) })
	err = os.Chmod(chrootDir, os.FileMode(0755))
	TestExpectSuccess(t, err)
	TestExpectSuccess(t, os.Mkdir(filepath.Join(chrootDir, "dev"), os.FileMode(0755)))

	request := [][]string{[]string{"CHROOT", chrootDir}}
	reply, err := MakeRequest(socket, request, 10*time.Second)
	TestExpectSuccess(t, err)
	TestEqual(t, reply, "REQUEST OK\n")

	// Next check to see that the init daemon is chrooted in the right
	// place.
	root, err := os.Readlink(fmt.Sprintf("/proc/%d/root", pid))
	TestExpectSuccess(t, err)
	TestEqual(t, root, chrootDir)
}

func TestBadChrootcRequest(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	tests := [][][]string{
		// Test 1: Request is too long.
		[][]string{
			[]string{"CHROOT", "DIR", "FALSE"},
			[]string{"EXTRA"},
		},

		// Test 2: Request is missing a directory.
		[][]string{
			[]string{"CHROOT"},
		},

		// Test 3: Extra cruft.
		[][]string{
			[]string{"CHROOT", "DIR", "FALSE", "EXTRA"},
		},
	}
	BadResultsCheck(t, tests)
}
