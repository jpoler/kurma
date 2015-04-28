// Copyright 2012-2014 Apcera Inc. All rights reserved.

// +build windows

package logray

// A compatibility mapping of colour names to ANSI escape sequences.  Entries
// here should match those used for the ANSI setup.  In future, if we want
// colour support on Windows, we will need to replace all of iooutput for
// Windows, as originally intended.
var ioOutputColorMap map[string][]byte = map[string][]byte{
	"default":        []byte(""),
	"black":          []byte(""),
	"brlight-black":  []byte(""),
	"red":            []byte(""),
	"bright-red":     []byte(""),
	"green":          []byte(""),
	"bright-green":   []byte(""),
	"yellow":         []byte(""),
	"bright-yellow":  []byte(""),
	"blue":           []byte(""),
	"bright-blue":    []byte(""),
	"magenta":        []byte(""),
	"bright-magenta": []byte(""),
	"cyan":           []byte(""),
	"bright-cyan":    []byte(""),
	"white":          []byte(""),
	"bright-white":   []byte(""),
	"crazy":          []byte(""),
	"half-crazy":     []byte(""),
}
