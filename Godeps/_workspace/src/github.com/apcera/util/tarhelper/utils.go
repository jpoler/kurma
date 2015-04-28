// Copyright 2014-2015 Apcera Inc. All rights reserved.

package tarhelper

import (
	"archive/tar"
	"bufio"
	"io"
)

// defaultMappingFunc is the default mapping function when taring or untaring
// without specifying your own mapping function.
func defaultMappingFunc(id int) (int, error) {
	return id, nil
}

// DetectArchiveCompression takes a source reader and will determine the
// compression type to use, if any. It will return a *tar.Reader that can be
// used to read the archive.
func DetectArchiveCompression(r io.Reader) (*tar.Reader, error) {
	var comp Decompressor

	// setup a buffered reader
	br := bufio.NewReader(r)

	// loop over the registered decompressors to find the right one
	for _, c := range decompressorTypes {
		if c.Detect(br) {
			comp = c
			break
		}
	}

	// Create the reader if a compression handler was found, else fall back on
	// using no compression.
	if comp != nil {
		// Create the reader
		arch, err := comp.NewReader(br)
		if err != nil {
			return nil, err
		}
		defer func() {
			if cl, ok := arch.(io.ReadCloser); ok {
				cl.Close()
			}
		}()
		return tar.NewReader(arch), nil
	}

	return tar.NewReader(br), nil
}
