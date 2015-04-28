// Copyright 2014 Apcera Inc. All rights reserved.

package tarhelper

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"io"
)

var decompressorTypes map[string]Decompressor

func AddDecompressor(name string, comp Decompressor) {
	decompressorTypes[name] = comp
}

func init() {
	decompressorTypes = map[string]Decompressor{}
	AddDecompressor("gzip", &GzipDecompressor{})
	AddDecompressor("bzip2", &Bzip2Decompressor{})
}

type Decompressor interface {
	Detect(*bufio.Reader) bool
	NewReader(io.Reader) (io.Reader, error)
}

type GzipDecompressor struct{}

func (c *GzipDecompressor) Detect(br *bufio.Reader) bool {
	data, err := br.Peek(2)
	if err != nil {
		return false
	}
	return bytes.Equal(data, []byte{0x1f, 0x8b})
}

func (c *GzipDecompressor) NewReader(src io.Reader) (io.Reader, error) {
	return gzip.NewReader(src)
}

type Bzip2Decompressor struct{}

func (c *Bzip2Decompressor) Detect(br *bufio.Reader) bool {
	data, err := br.Peek(2)
	if err != nil {
		return false
	}
	return bytes.Equal(data, []byte{0x42, 0x5a})
}

func (c *Bzip2Decompressor) NewReader(src io.Reader) (io.Reader, error) {
	return bzip2.NewReader(src), nil
}
