// Copyright 2013 Apcera Inc. All rights reserved.

package hashutil

import (
	"crypto/md5"
	"encoding/hex"
	"io"
)

// Simple function that allows verification that a string is in fact a valid MD5
// formated string.
func IsMd5Valid(s string) bool {
	if len(s) != 32 {
		return false
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return false
	}
	if len(b) != 16 {
		return false
	}
	return true
}

// Intermediate Reader object that will calculate the MD5 value of the data that
// passes through it.
type Md5Reader struct {
	*hashReader
}

// Returns a new Md5Reader.
func NewMd5(r io.Reader) *Md5Reader {
	m := &Md5Reader{}
	m.hashReader = newHashReader(md5.New(), r)
	return m
}

// Returns the MD5 for all data that has been passed through this Reader
// already. This should ideally be called after the Reader is Closed, but
// otherwise its safe to call any time.
func (m *Md5Reader) Md5() string {
	return hex.EncodeToString(m.hash.Sum(nil))
}
