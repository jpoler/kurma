// Copyright 2014 Apcera Inc. All rights reserved.

package xz

import (
	"bufio"
	"bytes"
	"io"

	"github.com/apcera/util/tarhelper"
	xz "github.com/remyoudompheng/go-liblzma"
)

func init() {
	tarhelper.AddDecompressor("xz", &XZDecompressor{})
}

type XZDecompressor struct{}

func (c *XZDecompressor) Detect(br *bufio.Reader) bool {
	data, err := br.Peek(6)
	if err != nil {
		return false
	}
	return bytes.Equal(data, []byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00})
}

func (c *XZDecompressor) NewReader(src io.Reader) (io.Reader, error) {
	return xz.NewReader(src)
}
