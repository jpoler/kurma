// Copyright 2015 Apcera Inc. All rights reserved.

package hashutil

import (
	"crypto/sha512"
	"encoding/hex"
	"io"
)

// Simple function that allows verification that a string is in fact a valid
// SHA512 formated string.
func IsSha512Valid(s string) bool {
	if len(s) != 128 {
		return false
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return false
	}
	if len(b) != 64 {
		return false
	}
	return true
}

// Intermediate Reader object that will calculate the SHA512 value of the data
// that passes through it.
type Sha512Reader struct {
	*hashReader
}

// Returns a new Sha512Reader.
func NewSha512(r io.Reader) *Sha512Reader {
	s := new(Sha512Reader)
	s.hashReader = newHashReader(sha512.New(), r)
	return s
}

// Returns the SHA512 for all data that has been passed through this Reader
// already. This should ideally be called after the Reader is Closed, but
// otherwise its safe to call any time.
func (s *Sha512Reader) Sha512() string {
	return hex.EncodeToString(s.hash.Sum(nil))
}
