// Copyright 2013 Apcera Inc. All rights reserved.

package testtool

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// -----------------------------------------------------------------------
// Equality tests.
// -----------------------------------------------------------------------

// This describes the given object and if the output is too long then it
// suppresses it unless --debug was used.
func describe(prefix string, i interface{}) string {
	out := fmt.Sprintf("%s%#v", prefix, i)
	if len(out) > 160 && !TestDebug {
		out = fmt.Sprintf(
			"%s: Value suppressed. Use --debug to see it.", prefix)
	}
	return out
}

// Returns true if the value is nil. Interfaces can actually NOT be nil since
// they have a type attached to them, even if the interface value is nil so
// we check both cases in this function.
func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	// If the value is a Kind which can store nil then we actually
	// check it, otherwise the IsNil() call can panic.
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Func:
	case reflect.Interface:
	case reflect.Map:
	case reflect.Ptr:
	case reflect.Slice:
	default:
		return false
	}
	return v.IsNil()
}

func TestExpectNonNil(t Logger, i interface{}, msg ...string) {
	if haveNil := isNil(i); haveNil {
		Fatalf(t, "Expected non-nil value, got nil. %s", msg)
	}
}

func TestTrue(t Logger, ans bool) {
	if !ans {
		Fatalf(t, "Expected a true value.")
	}
}

func TestFalse(t Logger, ans bool) {
	if ans {
		Fatalf(t, "Expected a false value.")
	}
}

func TestMatch(t Logger, have string, r *regexp.Regexp) {
	if !r.MatchString(have) {
		Fatalf(t, "Expected %s to match regexp %v.", have, r)
	}
}

func TestEqual(t Logger, have, want interface{}, msg ...string) {
	haveNil := isNil(have)
	wantNil := isNil(want)
	reason := ""
	if len(msg) > 0 {
		reason = ": " + strings.Join(msg, "")
	}
	if haveNil && wantNil {
		return
	} else if haveNil && !wantNil {
		Fatalf(t, "%sExpected non nil, got nil.", reason)
	} else if !haveNil && wantNil {
		Fatalf(t, "%sExpected nil, got non nil.", reason)
	}
	haveValue := reflect.ValueOf(have)
	wantValue := reflect.ValueOf(want)
	r := deepValueEqual("", haveValue, wantValue, make(map[uintptr]*visit))
	if len(r) != 0 {
		Fatalf(t, "Not Equal%s\n%s", reason, strings.Join(r, "\n"))
	}
}

func TestNotEqual(t Logger, have, want interface{}, msg ...string) {
	haveNil := isNil(have)
	wantNil := isNil(want)
	reason := ""
	if len(msg) > 0 {
		reason = ": " + strings.Join(msg, "")
	}
	if haveNil && wantNil {
		Fatalf(t, "%sEquality not expected, have=nil", reason)
	} else if haveNil || wantNil {
		return
	}
	haveValue := reflect.ValueOf(have)
	wantValue := reflect.ValueOf(want)
	r := deepValueEqual("", haveValue, wantValue, make(map[uintptr]*visit))
	if len(r) == 0 {
		Fatalf(t,
			"Equality not expected%s\n%s", reason, describe("have: ", have))
	}
}

// ---------
// Internals
// ---------

// Tracks access to specific pointers so we do not recurse.
type visit struct {
	a1   uintptr
	a2   uintptr
	typ  reflect.Type
	next *visit
}

// This is ripped directly from golang 1.1 and modified in order to
// make this a little more unit test friendly.
// Tests for deep equality using reflected types. The map argument tracks
// comparisons that have already been seen, which allows short circuiting on
// recursive types.
func deepValueEqual(description string, have, want reflect.Value, visited map[uintptr]*visit) (diffs []string) {
	if !want.IsValid() && !have.IsValid() {
		return nil
	} else if !want.IsValid() && have.IsValid() {
		// This is rare, not sure how to document this better.
		return []string{
			fmt.Sprintf("%s: have invalid object.", description),
		}
	} else if want.IsValid() && !have.IsValid() {
		// This is rare, not sure how to document this better.
		return []string{
			fmt.Sprintf("%s: wanted a valid object.", description),
		}
	} else if want.Type() != have.Type() {
		return []string{fmt.Sprintf(
			"%s: Not the same type, have: '%s', want: '%s'",
			description, have.Type(), want.Type())}
	}

	if want.CanAddr() && have.CanAddr() {
		addr1 := want.UnsafeAddr()
		addr2 := have.UnsafeAddr()
		if addr1 > addr2 {
			// Canonicalize order to reduce number of entries in visited.
			addr1, addr2 = addr2, addr1
		}

		// Short circuit if references are identical ...
		if addr1 == addr2 {
			return []string{}
		}

		// ... or already seen
		h := 17*addr1 + addr2
		seen := visited[h]
		typ := want.Type()
		for p := seen; p != nil; p = p.next {
			if p.a1 == addr1 && p.a2 == addr2 && p.typ == typ {
				return []string{}
			}
		}

		// Remember for later.
		visited[h] = &visit{addr1, addr2, typ, seen}
	}

	// Checks to see if one value is nil, while the other is not.
	checkNil := func() bool {
		if want.IsNil() && !have.IsNil() {
			diffs = append(diffs, fmt.Sprintf("%s: not equal.", description))
			diffs = append(diffs, describe("have: ", have.Interface()))
			diffs = append(diffs, "want: nil")
			return true
		} else if !want.IsNil() && have.IsNil() {
			diffs = append(diffs, fmt.Sprintf("%s: not equal.", description))
			diffs = append(diffs, "have: nil")
			diffs = append(diffs, describe("want: ", have.Interface()))
			return true
		}
		return false
	}

	// Checks to see that the lengths of both objects are equal.
	checkLen := func() bool {
		if want.Len() != have.Len() {
			diffs = append(diffs, fmt.Sprintf(
				"%s: (len(have): %d, len(want): %d)",
				description, have.Len(), want.Len()))
			diffs = append(diffs, describe("have: ", have.Interface()))
			diffs = append(diffs, describe("want: ", want.Interface()))
			return true
		}
		return false
	}

	switch want.Kind() {
	case reflect.Array:
		if !checkLen() {
			for i := 0; i < want.Len(); i++ {
				newdiffs := deepValueEqual(
					fmt.Sprintf("%s[%d]", description, i),
					have.Index(i), want.Index(i), visited)
				diffs = append(diffs, newdiffs...)
			}
		}

	case reflect.Slice:
		if !checkNil() && !checkLen() {
			for i := 0; i < want.Len(); i++ {
				newdiffs := deepValueEqual(
					fmt.Sprintf("%s[%d]", description, i),
					have.Index(i), want.Index(i), visited)
				diffs = append(diffs, newdiffs...)
			}
		}

	case reflect.Interface:
		if !checkNil() {
			newdiffs := deepValueEqual(description, have.Elem(), want.Elem(), visited)
			diffs = append(diffs, newdiffs...)
		}

	case reflect.Ptr:
		newdiffs := deepValueEqual(description, have.Elem(), want.Elem(), visited)
		diffs = append(diffs, newdiffs...)

	case reflect.Struct:
		for i, n := 0, want.NumField(); i < n; i++ {
			f := want.Type().Field(i)
			if len(f.PkgPath) != 0 {
				// skip unexported fields
				continue
			}
			name := f.Name
			// Make sure that we don't print a strange error if the
			// first object given to us is a struct.
			if description == "" {
				newdiffs := deepValueEqual(
					name, have.Field(i), want.Field(i), visited)
				diffs = append(diffs, newdiffs...)
			} else {
				newdiffs := deepValueEqual(
					fmt.Sprintf("%s.%s", description, name),
					have.Field(i), want.Field(i), visited)
				diffs = append(diffs, newdiffs...)
			}
		}

	case reflect.Map:
		if !checkNil() {
			// Check that the keys are present in both maps.
			for _, k := range want.MapKeys() {
				if !have.MapIndex(k).IsValid() {
					// Add the error.
					diffs = append(diffs, fmt.Sprintf(
						"%sExpected key [%q] is missing.", description, k))
					diffs = append(diffs, "have: not present")
					diffs = append(diffs,
						describe("want: ", want.MapIndex(k).Interface()))
					continue
				}
				newdiffs := deepValueEqual(
					fmt.Sprintf("%s[%q] ", description, k),
					have.MapIndex(k), want.MapIndex(k), visited)
				diffs = append(diffs, newdiffs...)
			}
			for _, k := range have.MapKeys() {
				if !want.MapIndex(k).IsValid() {
					// Add the error.
					diffs = append(diffs, fmt.Sprintf("%sUnexpected key [%q].", description, k))
					diffs = append(diffs, describe("have: ", have.MapIndex(k).Interface()))
					diffs = append(diffs, "want: not present")
				}
			}
		}

	case reflect.Func:
		// Can't do better than this:
		checkNil()

	case reflect.String:
		// We know the underlying type is a string so calling String()
		// will return the underlying value. Trying to call Interface()
		// and assert to a string will panic.
		s1 := have.String()
		s2 := want.String()
		if len(s1) != len(s2) {
			return []string{
				fmt.Sprintf(
					"%s: len(have) %d != len(want) %d.",
					description, len(s1), len(s2)),
				describe("have: ", s1),
				describe("want: ", s2),
			}
		}
		for i := range s1 {
			if s1[i] != s2[i] {
				return []string{
					fmt.Sprintf(
						"%s: difference at index %d.",
						description, i),
					describe("have: ", s1),
					describe("want: ", s2),
				}
			}
		}

	default:
		// Specific low level types:
		switch want.Interface().(type) {
		case bool:
			s1 := have.Interface().(bool)
			s2 := want.Interface().(bool)
			if s1 != s2 {
				return []string{fmt.Sprintf("%s: have %t, want %t", description, s1, s2)}
			}
		case int:
			s1 := have.Interface().(int)
			s2 := want.Interface().(int)
			if s1 != s2 {
				return []string{fmt.Sprintf("%s: have %d, want %d", description, s1, s2)}
			}
		case int8:
			s1 := have.Interface().(int8)
			s2 := want.Interface().(int8)
			if s1 != s2 {
				return []string{fmt.Sprintf("%s: have %d, want %d", description, s1, s2)}
			}
		case int16:
			s1 := have.Interface().(int16)
			s2 := want.Interface().(int16)
			if s1 != s2 {
				return []string{fmt.Sprintf("%s: have %d, want %d", description, s1, s2)}
			}
		case int32:
			s1 := have.Interface().(int32)
			s2 := want.Interface().(int32)
			if s1 != s2 {
				return []string{fmt.Sprintf("%s: have %d, want %d", description, s1, s2)}
			}
		case int64:
			s1 := have.Interface().(int64)
			s2 := want.Interface().(int64)
			if s1 != s2 {
				return []string{fmt.Sprintf("%s: have %d, want %d", description, s1, s2)}
			}
		case uint:
			s1 := have.Interface().(uint)
			s2 := want.Interface().(uint)
			if s1 != s2 {
				return []string{fmt.Sprintf("%s: have %d, want %d", description, s1, s2)}
			}
		case uint8:
			s1 := have.Interface().(uint8)
			s2 := want.Interface().(uint8)
			if s1 != s2 {
				return []string{fmt.Sprintf("%s: have %d, want %d", description, s1, s2)}
			}
		case uint16:
			s1 := have.Interface().(uint16)
			s2 := want.Interface().(uint16)
			if s1 != s2 {
				return []string{fmt.Sprintf("%s: have %d, want %d", description, s1, s2)}
			}
		case uint32:
			s1 := have.Interface().(uint32)
			s2 := want.Interface().(uint32)
			if s1 != s2 {
				return []string{fmt.Sprintf("%s: have %d, want %d", description, s1, s2)}
			}
		case uint64:
			s1 := have.Interface().(uint64)
			s2 := want.Interface().(uint64)
			if s1 != s2 {
				return []string{fmt.Sprintf("%s: have %d, want %d", description, s1, s2)}
			}
		case uintptr:
			s1 := have.Interface().(uintptr)
			s2 := want.Interface().(uintptr)
			if s1 != s2 {
				return []string{fmt.Sprintf("%s: have %d, want %d", description, s1, s2)}
			}
		case float32:
			s1 := have.Interface().(float32)
			s2 := want.Interface().(float32)
			if s1 != s2 {
				return []string{fmt.Sprintf("%s: have %f, want %f", description, s1, s2)}
			}
		case float64:
			s1 := have.Interface().(float64)
			s2 := want.Interface().(float64)
			if s1 != s2 {
				return []string{fmt.Sprintf("%s: have %f, want %f", description, s1, s2)}
			}
		default:
			// Normal equality suffices
			if !reflect.DeepEqual(want.Interface(), have.Interface()) {
				return []string{
					fmt.Sprintf("%s: not equal.", description),
					describe("have: ", have),
					describe("want: ", want),
				}
			}
		}
	}

	// This shouldn't ever be reached.
	return diffs
}
