// Copyright 2012 Apcera Inc. All rights reserved.

package str

import (
	"testing"
)

func checkFmt(t *testing.T, v int64, expected string) {
	s := FormatIntv(v)
	if s != expected {
		t.Fatalf("Formatting %d generated '%s' when expected '%s'",
			v, s, expected)
	}
}

func checkFmtv(t *testing.T, v interface{}, expected string) {
	s := FormatIntv(v)
	if s != expected {
		t.Fatalf("Formatting %d generated '%s' when expected '%s'",
			v, s, expected)
	}
}

func TestFormatInt(t *testing.T) {

	checkFmt(t, int64(0), "0")
	checkFmt(t, int64(10), "10")
	checkFmt(t, int64(100), "100")
	checkFmt(t, int64(999), "999")
	checkFmt(t, int64(1000), "1,000")
	checkFmt(t, int64(10000), "10,000")
	checkFmt(t, int64(100000), "100,000")
	checkFmt(t, int64(1000000), "1,000,000")
	checkFmt(t, int64(10000000), "10,000,000")
	checkFmt(t, int64(100000000), "100,000,000")
	checkFmt(t, int64(1000000000), "1,000,000,000")

	checkFmt(t, int64(-10), "-10")
	checkFmt(t, int64(-100), "-100")
	checkFmt(t, int64(-999), "-999")
	checkFmt(t, int64(-1000), "-1,000")
	checkFmt(t, int64(-10000), "-10,000")
	checkFmt(t, int64(-100000), "-100,000")
	checkFmt(t, int64(-1000000), "-1,000,000")
	checkFmt(t, int64(-10000000), "-10,000,000")
	checkFmt(t, int64(-100000000), "-100,000,000")
	checkFmt(t, int64(-1000000000), "-1,000,000,000")

	// Check min/max

	var i64 int64
	var u64 uint64

	i64 = 9223372036854775807
	checkFmtv(t, i64, "9,223,372,036,854,775,807")

	i64 = -9223372036854775808
	checkFmtv(t, i64, "-9,223,372,036,854,775,808")

	u64 = 18446744073709551615
	checkFmtv(t, u64, "18,446,744,073,709,551,615")

}
