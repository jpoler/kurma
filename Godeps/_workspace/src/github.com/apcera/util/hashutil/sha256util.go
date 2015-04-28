// Copyright 2013 Apcera Inc. All rights reserved.

package hashutil

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// Simple function that allows verification that a string is in fact a valid
// SHA256 formated string.
func IsSha256Valid(s string) bool {
	if len(s) != 64 {
		return false
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return false
	}
	if len(b) != 32 {
		return false
	}
	return true
}

// Intermediate Reader object that will calculate the SHA256 value of the data
// that passes through it.
type Sha256Reader struct {
	*hashReader
}

// Returns a new Sha256Reader.
func NewSha256(r io.Reader) *Sha256Reader {
	s := new(Sha256Reader)
	s.hashReader = newHashReader(sha256.New(), r)
	return s
}

// Returns the SHA256 for all data that has been passed through this Reader
// already. This should ideally be called after the Reader is Closed, but
// otherwise its safe to call any time.
func (s *Sha256Reader) Sha256() string {
	return hex.EncodeToString(s.hash.Sum(nil))
}
