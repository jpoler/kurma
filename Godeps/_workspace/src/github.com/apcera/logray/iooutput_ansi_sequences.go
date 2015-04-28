// Copyright 2012-2014 Apcera Inc. All rights reserved.

// +build !windows

package logray

// A basic mapping of color name to ANSI bytes.  Entries here should also exist
// in the Windows variant.
var ioOutputColorMap map[string][]byte = map[string][]byte{
	"default":        []byte("\033[0m"),
	"black":          []byte("\033[0;30m"),
	"brlight-black":  []byte("\033[0;30;1m"),
	"red":            []byte("\033[0;31m"),
	"bright-red":     []byte("\033[0;31;1m"),
	"green":          []byte("\033[0;32m"),
	"bright-green":   []byte("\033[0;32;1m"),
	"yellow":         []byte("\033[0;33m"),
	"bright-yellow":  []byte("\033[0;33;1m"),
	"blue":           []byte("\033[0;34m"),
	"bright-blue":    []byte("\033[0;34;1m"),
	"magenta":        []byte("\033[0;35m"),
	"bright-magenta": []byte("\033[0;35;1m"),
	"cyan":           []byte("\033[0;36m"),
	"bright-cyan":    []byte("\033[0;36;1m"),
	"white":          []byte("\033[0;37m"),
	"bright-white":   []byte("\033[0;37;1m"),
	"crazy":          []byte("\033[40;1;35m"),
	"half-crazy":     []byte("\033[40;1;33m"),
}
