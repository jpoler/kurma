// Copyright 2013 Apcera Inc. All rights reserved.

package proc

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// ReadInt64 reads one int64 number from the first line of a file.
func ReadInt64(file string) (int64, error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	buf := make([]byte, 19)
	n, err := f.Read(buf)
	if err != nil {
		return 0, err
	}

	p := strings.Split(string(buf[0:n]), "\n")
	v, err := strconv.ParseInt(p[0], 10, 64)
	if err != nil {
		return 0, err
	}

	return v, nil
}

// Parses the given file into various elements. This function assumes basic
// white space semantics (' ' and '\t' for column splitting, and '\n' for
// row splitting.
func ParseSimpleProcFile(
	filename string,
	lf func(index int, line string) error,
	ef func(line, index int, elm string) error) error {

	fd, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fd.Close()

	contentsBytes, err := ioutil.ReadAll(fd)
	if err != nil {
		return err
	}

	// Setup base handlers if they were passed in as nil.
	if lf == nil {
		lf = func(index int, line string) error { return nil }
	}
	if ef == nil {
		ef = func(line, index int, elm string) error { return nil }
	}

	contents := string(contentsBytes)
	lines := strings.Split(contents, "\n")

	for li, l := range lines {
		for ei, e := range strings.Fields(l) {
			if err := ef(li, ei, e); err != nil {
				return err
			}
		}
		if err := lf(li, l); err != nil {
			return err
		}
	}

	return nil
}
