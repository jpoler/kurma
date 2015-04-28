// Copyright 2013 Apcera Inc. All rights reserved.

package envmap

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	tt "github.com/apcera/util/testtool"
)

func TestEnvMapSimple(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	// Simple case.
	e := NewEnvMap()
	e.Set("A", "$A")
	m := e.Map()
	if len(m) != 1 {
		tt.Fatalf(t, "Invlid number of environment variables set.")
	} else if v, ok := m["A"]; ok == false {
		tt.Fatalf(t, "$A was not defined.")
	} else if v != "" {
		tt.Fatalf(t, "$A has a bad value: %s", v)
	}
}

func TestEnvMapRecursive(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	// Simple case.
	e := NewEnvMap()
	e.Set("A", "$B")
	e.Set("B", "$A")
	m := e.Map()
	if len(m) != 2 {
		tt.Fatalf(t, "Invalid number of environment variables set.")
	} else if v, ok := m["A"]; ok == false {
		tt.Fatalf(t, "$A was not defined.")
	} else if v != "" {
		tt.Fatalf(t, "$A has a bad vaule: %s", v)
	} else if v, ok := m["B"]; ok == false {
		tt.Fatalf(t, "$B was not defined.")
	} else if v != "" {
		tt.Fatalf(t, "$B has a bad vaule: %s", v)
	}
}

func TestEnvMapDoubleAdd(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	// Simple case.
	e := NewEnvMap()
	e.Set("A", "1$A")
	e.Set("A", "2$A")
	m := e.Map()
	if len(m) != 1 {
		tt.Fatalf(t, "Invlid number of environment variables set.")
	} else if v, ok := m["A"]; ok == false {
		tt.Fatalf(t, "$A was not defined.")
	} else if v != "21" {
		tt.Fatalf(t, "$A has a bad vaule: %s", v)
	}
}

func TestEnvMap(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	expected := []string{
		"START_COMMAND=XYZ",
		"TEST_CROSSREF1=cross_pkg2",
		"TEST_CROSSREF2=cross_pkg1",
		"TEST_CROSSREF_VAL1=cross_pkg1",
		"TEST_CROSSREF_VAL2=cross_pkg2",
		"TEST_ENV_VARIABLE=TEST_ENV_CONTENT",
		"TEST_MERGE_UNDEFINED=undef:",
		"TEST_MERGE_VARIABLE=pkg2:pkg1:",
		"TEST_OVERRIDE_VARIABLE=pkg2",
	}

	root := NewEnvMap()
	root.Set("TEST_ENV_VARIABLE", "TEST_ENV_CONTENT")
	root.Set("TEST_MERGE_VARIABLE", "pkg1:$TEST_MERGE_VARIABLE")
	root.Set("TEST_MERGE_UNDEFINED", "undef:$TEST_NOT_SET")
	root.Set("TEST_OVERRIDE_VARIABLE", "pkg1")
	root.Set("TEST_CROSSREF1", "$TEST_CROSSREF_VAL2")
	root.Set("TEST_CROSSREF_VAL1", "cross_pkg1")
	root.Set("START_COMMAND", "XYZ")

	c1 := root.NewChild()
	c1.Set("TEST_MERGE_VARIABLE", "pkg2:$TEST_MERGE_VARIABLE")
	c1.Set("TEST_OVERRIDE_VARIABLE", "pkg2")
	c1.Set("TEST_CROSSREF2", "$TEST_CROSSREF_VAL1")
	c1.Set("TEST_CROSSREF_VAL2", "cross_pkg2")
	c1.Set("START_COMMAND", "XYZ")

	envstrs := c1.Strings()

	// Sort the two list just in case.
	sort.Sort(sort.StringSlice(envstrs))
	sort.Sort(sort.StringSlice(expected))

	msg := make([]string, 0, len(envstrs))
	failed := false
	a := func(fmtstr string, args ...interface{}) {
		msg = append(msg, fmt.Sprintf(fmtstr, args...))
	}
	for i := 0; i < len(expected) || i < len(envstrs); i++ {
		if i >= len(expected) {
			a("\t'' > '%s'", envstrs[i])
		} else if i >= len(envstrs) {
			a("\t'%s' < ''", expected[i])
		} else if expected[i] != envstrs[i] {
			a("\t'%s' != '%s'", expected[i], envstrs[i])
		} else {
			a("\t'%s' == '%s'", expected[i], envstrs[i])
			continue
		}
		failed = true
	}
	if failed == true {
		tt.Fatalf(t, "results are not the same:\n%s", strings.Join(msg, "\n"))
	}
}

func TestEnvMapGetRaw(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	root := NewEnvMap()
	root.Set("VARIABLE", "foo bar $STR")
	root.Set("VAR1", "abc 123 $VAR2")
	root.Set("VAR2", "xyz")

	if v, _ := root.Get("VARIABLE"); v != "foo bar " {
		tt.Fatalf(t, "Get(VARIABLE) should have blanked out STR: %q", v)
	}
	if v, _ := root.GetRaw("VARIABLE"); v != "foo bar $STR" {
		tt.Fatalf(t, "GetRaw(VARIABLE) should include raw $STR: %q", v)
	}

	if v, _ := root.Get("VAR1"); v != "abc 123 xyz" {
		tt.Fatalf(t, "Get(VAR1) should have parsed mentioned variables: %q", v)
	}
	if v, _ := root.GetRaw("VAR1"); v != "abc 123 $VAR2" {
		tt.Fatalf(t, "GetRaw(VAR1) should be the raw string: %q", v)
	}
}
func TestEnvMapGetUnflattened(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	root := NewEnvMap()
	root.Set("VARIABLE", "foo bar $STR")
	root.Set("VAR1", "abc 123 $VAR2")
	root.Set("VAR2", "xyz")

	root.FlattenMap(false)
	envMap := root.Map()

	tt.TestEqual(t, envMap["VAR1"], "abc 123 $VAR2")
	tt.TestEqual(t, envMap["VAR2"], "xyz")
	tt.TestEqual(t, envMap["VARIABLE"], "foo bar $STR")
}
