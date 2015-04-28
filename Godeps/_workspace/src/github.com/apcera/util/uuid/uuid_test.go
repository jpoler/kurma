// Copyright 2012,2013 Apcera Inc. All rights reserved.

package uuid_test

import (
	"fmt"
	"github.com/apcera/util/uuid"
	"testing"
)

// Documents a simple use case for a generating and comparing UUIDs.
func Example() {
	u1 := uuid.Variant1()
	u2 := uuid.Variant4()
	fmt.Println("Text representation of a uuid: ", u1.String())
	if u1.Equal(u2) {
		fmt.Println("UUIDs shouldn't ever be equal so this is not reached.")
	}
}

// Verifies that Variant1() never generates duplicate UUIDs. This is not so much
// an absolute proof as it is a simple check to ensure that basic functionality
// is not broken. This will catch very basic screw ups and is easy to implement.
func TestVariant1(t *testing.T) {
	previous := make(map[string]bool)

	for i := 0; i < 10000; i++ {
		u := uuid.Variant1().String()
		if _, exists := previous[u]; exists == true {
			t.Fatal("Duplicate UUIDs generated from Variant1(): ", u)
		}
		previous[u] = true
	}
}

// Verifies that Variant3() produces consistent results for a given name space
// and name.
func TestVariant3(t *testing.T) {
	same1 := uuid.Variant3(uuid.NameSpaceDNS(), "some-name")
	same2 := uuid.Variant3(uuid.NameSpaceDNS(), "some-name")
	other_name := uuid.Variant3(uuid.NameSpaceDNS(), "some-other-name")
	other_domain := uuid.Variant3(uuid.NameSpaceX500(), "some-name")

	if same1.String() != "b43b195b-cea8-388b-84c6-453be0976081" {
		t.Fatal("UUID generation for 'some-name' doesn't match known output.")
	}

	if !same1.Equal(same2) {
		t.Fatal("UUID generation for 'some-name' is not the same.")
	}

	if same1.Equal(other_name) {
		t.Fatal("UUID generation for 'some-other-name' were the same.")
	}

	if same1.Equal(other_domain) {
		t.Fatal("UUID generation for 'some-name' in other domain not different..")
	}
}

// Tests that Variant4() will not produce duplicate UUIDs. This is actually not
// assured by the protocol given that its 125 bits of random numbers. Still,
// the likely hood of duplicating in a few thousand tries should be low enough
// 2^125 / 1000
func TestVariant4(t *testing.T) {
	previous := make(map[string]bool)

	for i := 0; i < 10000; i++ {
		u := uuid.Variant4().String()
		if _, exists := previous[u]; exists == true {
			t.Fatal("Duplicate UUIDs generated from Variant4(): ", u)
		}
		previous[u] = true
	}
}

// Verifies that Variant5() produces consistent results for a given name space
// and name.
func TestVariant5(t *testing.T) {
	same1 := uuid.Variant5(uuid.NameSpaceDNS(), "some-name")
	same2 := uuid.Variant5(uuid.NameSpaceDNS(), "some-name")
	other_name := uuid.Variant5(uuid.NameSpaceDNS(), "some-other-name")
	other_domain := uuid.Variant5(uuid.NameSpaceX500(), "some-name")

	if same1.String() != "33d1b922-4cf2-566d-a20a-bc0f77f14de7" {
		t.Fatal("UUID generation for 'some-name' doesn't match known output.")
	}

	if !same1.Equal(same2) {
		t.Fatal("UUID generation for 'some-name' is not the same.")
	}

	if same1.Equal(other_name) {
		t.Fatal("UUID generation for 'some-other-name' were the same.")
	}

	if same1.Equal(other_domain) {
		t.Fatal("UUID generation for 'some-name' in other domain not different..")
	}
}

// Test to make sure that FromString works by testing a good string as well
// as a few bad ones.
func TestFromString(t *testing.T) {
	s_valid := "01234567-890a-1bcd-af01-234567890abc"
	if _, err := uuid.FromString(s_valid); err != nil {
		t.Fatal("Failed to parse a valid UUID: " + s_valid + " error: " +
			err.Error())
	}

	s_invalid_len := "123456789abcdef"
	if _, err := uuid.FromString(s_invalid_len); err == nil {
		t.Fatal("Failed to detect a short UUID: " + s_invalid_len)
	}

	s_invalid_dash := "01234567f890af1bcdfef01f234567890abc"
	if _, err := uuid.FromString(s_invalid_dash); err == nil {
		t.Fatal("Failed to detect an invalid UUID: " + s_invalid_dash)
	}

	s_reserved_bits := "01234567-890a-1bcd-ef01-234567890abc"
	if _, err := uuid.FromString(s_reserved_bits); err == nil {
		t.Fatal("Failed to detect an invalid UUID: " + s_invalid_dash)
	}
}

// End to end test, generate, then convert to string, then UUID and back and
// verify that it comes out looking correct.
func TestEndToEndConversion(t *testing.T) {
	initial := uuid.Variant1()
	to_string := initial.String()
	from_string, err := uuid.FromString(to_string)
	if err != nil {
		t.Fatal("Parsing error for an unknown reason: " + err.Error())
	}

	if !initial.Equal(from_string) {
		t.Fatal("Result of Variant1() -> String() -> FromString are not equal.")
	}
}

// Benchmarks the Variant1() UUIDs.
func BenchmarkVariant1(b *testing.B) {
	// Generate the first UUID away from the benchmark timer so we can get
	// the mac address detection out of the way before the benchmark starts.
	b.StopTimer()
	uuid.Variant1()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		uuid.Variant1()
	}
}

// Benchmarks the Variant3() UUIDs.
func BenchmarkVariant3(b *testing.B) {
	u := uuid.NameSpaceURL()
	for i := 0; i < b.N; i++ {
		uuid.Variant3(u, "http://apcera.com/uuidtest_url_demo_url")
	}
}

// Benchmarks the Variant4() UUIDs.
func BenchmarkVariant4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		uuid.Variant4()
	}
}

// Benchmarks the Variant5() UUIDs.
func BenchmarkVariant5(b *testing.B) {
	u := uuid.NameSpaceURL()
	for i := 0; i < b.N; i++ {
		uuid.Variant5(u, "http://apcera.com/uuidtest_url_demo_url")
	}
}

// Benchmark the default string formatting function.
func BenchmarkString(b *testing.B) {
	// Generate a UUID outside of the timer.
	b.StopTimer()
	u := uuid.Variant1()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		u.String()
	}
}

// Benchmark the FromString() function.
func BenchmarkFromString(b *testing.B) {
	// Generate a UUID outside of the timer.
	b.StopTimer()
	u := uuid.Variant1().String()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		uuid.FromString(u)
	}
}

// This function is not to be used, but instead is intended to demonstrate the
// performance of the default approach (fmt.Sprintf). This is included in the
// test files in order to ensure that it doesn't get used in production code.
func stringSlowSprintf(u uuid.UUID) (r string) {
	fmt.Sprintf(r, "%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-"+
		"%02x%02x%02x%02x%02x%02x",
		u[0], u[1], u[2], u[3], u[4], u[5], u[6], u[7],
		u[8], u[9], u[10], u[11], u[12], u[13], u[14], u[15])
	return r
}

// Benchmark the fmt.Sprintf() based string function. This is simply to
// demonstrate the relative speed of this approach vs the slightly janky
// but much faster approach taken in the default String() function.
func BenchmarkStringSlowSprintf(b *testing.B) {
	// Generate a UUID outside of the timer.
	b.StopTimer()
	u := uuid.Variant1()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		stringSlowSprintf(u)
	}
}
