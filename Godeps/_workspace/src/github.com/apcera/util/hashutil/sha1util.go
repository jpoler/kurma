// Copyright 2013 Apcera Inc. All rights reserved.

package hashutil

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
)

// Simple function that allows verification that a string is in fact a valid
// sha1 formated string.
func IsSha1Valid(s string) bool {
	if len(s) != 40 {
		return false
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return false
	}
	if len(b) != 20 {
		return false
	}
	return true
}

// Intermediate Reader object that will calculate the SHA1 value of the data
// that passes through it.
type Sha1Reader struct {
	*hashReader
}

// Returns a new Sha1Reader.
func NewSha1(r io.Reader) *Sha1Reader {
	s := new(Sha1Reader)
	s.hashReader = newHashReader(sha1.New(), r)
	return s
}

// Returns the SHA1 for all data that has been passed through this Reader
// already. This should ideally be called after the Reader is Closed, but
// otherwise its safe to call any time.
func (s *Sha1Reader) Sha1() string {
	return hex.EncodeToString(s.hash.Sum(nil))
}
