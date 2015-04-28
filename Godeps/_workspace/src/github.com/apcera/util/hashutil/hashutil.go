// Copyright 2013 Apcera Inc. All rights reserved.

package hashutil

import (
	"hash"
	"io"
)

// Intermediate Reader object that will calculate the checksum value of the data
// that passes through it.
type hashReader struct {
	// The io.Reader source.
	source io.Reader

	// The total length of the data.
	length int64

	// The intermediate hash that is created and passed to newHashReader.
	hash hash.Hash
}

// Returns a new hashReader.
func newHashReader(h hash.Hash, r io.Reader) *hashReader {
	m := &hashReader{
		source: r,
		hash:   h,
	}
	return m
}

// Reads from the source, and returns the values upstream after adding the data
// to the hash.
func (m *hashReader) Read(p []byte) (n int, err error) {
	n, err = m.source.Read(p)
	if n > 0 {
		// hash.Hash assures us that this can never return an error.
		m.hash.Write(p[0:n])
		m.length += int64(n)
	}
	return
}

// Closes the source, if it implements io.Closer.
func (m *hashReader) Close() error {
	if m.source == nil {
		return nil
	}
	if r, ok := m.source.(io.Closer); ok {
		return r.Close()
	}
	return nil
}

// Returns the number of bytes passed through this reader.
func (m *hashReader) Length() int64 {
	return m.length
}
