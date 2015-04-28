// Copyright 2013 Apcera Inc. All rights reserved.

package proc

import (
	"fmt"
	"strings"
	"testing"

	tt "github.com/apcera/util/testtool"
)

func TestReadInt64(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	f := tt.WriteTempFile(t, "foo\nbar")

	_, err := ReadInt64(f)
	tt.TestExpectError(t, err)

	f = tt.WriteTempFile(t, "123\n456")

	v, err := ReadInt64(f)
	tt.TestExpectSuccess(t, err)
	tt.TestEqual(t, v, int64(123))

	f = tt.WriteTempFile(t, "123456789")
	v, err = ReadInt64(f)
	tt.TestExpectSuccess(t, err)
	tt.TestEqual(t, v, int64(123456789))

	maxInt64 := fmt.Sprintf("%d", int64(1<<63-1))
	f = tt.WriteTempFile(t, maxInt64)

	v, err = ReadInt64(f)
	tt.TestExpectSuccess(t, err)
	tt.TestEqual(t, v, int64(1<<63-1))

	maxInt64WithExtra := fmt.Sprintf("%d666", int64(1<<63-1))
	f = tt.WriteTempFile(t, maxInt64WithExtra)

	v, err = ReadInt64(f)
	tt.TestExpectSuccess(t, err)
	tt.TestEqual(t, v, int64(1<<63-1))
}

func TestParseSimpleProcFile(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	// Test 1: Success.
	lines := []string{
		"aelm0 aelm1\taelm2",
		" belm0  belm1\t belm2\t\t\t",
		"",
		"delm0"}
	f := tt.WriteTempFile(t, strings.Join(lines, "\n"))
	err := ParseSimpleProcFile(
		f,
		func(index int, line string) error {
			if index > len(lines) {
				tt.Fatalf(t, "Too many lines read: %d", index)
			} else if line != lines[index] {
				tt.Fatalf(t, "Invalid line read: %s", line)
			}
			return nil
		},
		func(line int, index int, elm string) error {
			switch {
			case line == 0 && index == 0 && elm == "aelm0":
			case line == 0 && index == 1 && elm == "aelm1":
			case line == 0 && index == 2 && elm == "aelm2":
			case line == 1 && index == 0 && elm == "belm0":
			case line == 1 && index == 1 && elm == "belm1":
			case line == 1 && index == 2 && elm == "belm2":
			case line == 3 && index == 0 && elm == "delm0":
			default:
				tt.Fatalf(
					t, "Unknown element read: %d, %d, %s", line, index, elm)
			}
			return nil
		})
	if err != nil {
		tt.Fatalf(t, "Unexpected error from ParseSimpleProcFile()")
	}

	// Test 2: No function defined. This should be successful.
	err = ParseSimpleProcFile(f, nil, nil)
	if err != nil {
		tt.Fatalf(t, "Unexpected error from ParseSimpleProcFile()")
	}

	// Test 3: ef returns an error.
	err = ParseSimpleProcFile(
		f,
		func(index int, line string) error {
			return fmt.Errorf("error.")
		},
		nil)
	if err == nil {
		tt.Fatalf(t, "Expected error not returned.")
	}

	// Test 4: lf returns an error.
	err = ParseSimpleProcFile(
		f,
		nil,
		func(line int, index int, elm string) error {
			return fmt.Errorf("error.")
		})
	if err == nil {
		tt.Fatalf(t, "Expected error not returned.")
	}

	// Test 6: last case lf operation.
	err = ParseSimpleProcFile(
		f,
		func(index int, line string) error {
			if line == "delm0" {
				return fmt.Errorf("error")
			}
			return nil
		},
		nil)

	// Test 5: last case lf operation.
	err = ParseSimpleProcFile(
		f,
		nil,
		func(line int, index int, elm string) error {
			if elm == "delm0" {
				return fmt.Errorf("error")
			}
			return nil
		})
}
