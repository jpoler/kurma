// Copyright 2012 Apcera Inc. All rights reserved.

// Miscellaneous utilities formatting, coloring terminal output, and similar.
package str

import (
	"strconv"
)

func FormatInt(i int) string {
	if i >= 0 {
		return fmtImpl(uint64(i), false)
	}
	return fmtImpl(uint64(-i), true)
}

func FormatIntv(v interface{}) string {
	var i uint64
	var neg bool

	switch v.(type) {
	case int:
		in := v.(int)
		if in < 0 {
			i = uint64(-in)
			neg = true
		} else {
			i = uint64(in)
		}
	case int8:
		i8 := v.(int8)
		if i8 < 0 {
			i = uint64(-i8)
			neg = true
		} else {
			i = uint64(i8)
		}
	case int16:
		i16 := v.(int16)
		if i16 < 0 {
			i = uint64(-i16)
			neg = true
		} else {
			i = uint64(i16)
		}
	case int32:
		i32 := v.(int64)
		if i32 < 0 {
			i = uint64(-i32)
			neg = true
		} else {
			i = uint64(i32)
		}
	case int64:
		i64 := v.(int64)
		if i64 < 0 {
			i = uint64(-i64)
			neg = true
		} else {
			i = uint64(i64)
		}
	case uint:
		i = uint64(v.(uint))
	case uint8:
		i = uint64(v.(uint8))
	case uint16:
		i = uint64(v.(uint16))
	case uint32:
		i = uint64(v.(uint32))
	case uint64:
		i = v.(uint64)
	default:
		return "<invalid value in FormatIntv>"
	}

	return fmtImpl(i, neg)
}

func fmtImpl(i uint64, negative bool) string {
	s := strconv.FormatUint(i, 10)
	ex := 0
	if negative == true {
		ex = 1
	}
	n := len(s)
	// "g" is number of commas to insert, total output len is ex+g+len(s)
	g := (n - 1) / 3
	if g == 0 {
		if negative {
			return "-" + s
		}
		return s
	}
	buffer := make([]byte, 0, ex+g+len(s))
	if negative == true {
		buffer = append(buffer, '-')
	}
	// set "n" to length of head before first ","
	if n = n - g*3; n > 0 {
		buffer = append(buffer, s[:n]...)
	}
	for i := 0; i < g; i++ {
		buffer = append(buffer, byte(','))
		buffer = append(buffer, s[n:n+3]...)
		n += 3
	}
	return string(buffer)
}
