// Copyright 2013 Apcera Inc. All rights reserved.

package hashutil

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

type testStrings struct {
	num   int
	input string
	sha1  string
}

func TestSha1Reader(t *testing.T) {
	// Setup an array of testStrings
	tests := make([]testStrings, 0, 100)

	add := func(i string, s string) {
		tests = append(tests, testStrings{len(tests), i, s})
	}

	// A pair of simple tests.
	add("test1", "b444ac06613fc8d63795be9ad0beaf55011936ac")
	add("test2", "109f4b3c50d7b0df729d299bc6f8e9ef9066971f")

	// A long string test that makes sure that many Reads() still works.
	longStr := make([]byte, 1024*1024)
	for i := range longStr {
		longStr[i] = 'a'
	}
	add(string(longStr), "482c2b6d0089026a36845a8ff6a63757790f9906")

	// A zero byte string.
	add("", "da39a3ee5e6b4b0d3255bfef95601890afd80709")

	for i := range tests {
		r := NewSha1(strings.NewReader(tests[i].input))
		b := bufio.NewReader(r)
		out, err := b.ReadString(0)
		if err != nil && err != io.EOF {
			t.Fatalf("Error in test %d: %s", tests[i].num, err)
		} else if out != tests[i].input {
			t.Fatal("Read back different data.")
		}
	}
}

// Infinite reader. Returns results as fast as possible.
type infiniteReader struct {
}

// Does nothing but return the length of p.
func (i *infiniteReader) Read(p []byte) (int, error) {
	return len(p), nil
}

func benchmarkWrapper(blockSize int, b *testing.B) {
	buffer := make([]byte, blockSize)
	r := new(infiniteReader)
	s := NewSha1(r)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		n, _ := s.Read(buffer)
		b.SetBytes(int64(n))
	}
}

func BenchmarkSha1Reader128B(b *testing.B) {
	benchmarkWrapper(128, b)
}

func BenchmarkSha1Reader1K(b *testing.B) {
	benchmarkWrapper(1024, b)
}

func BenchmarkSha1Reader4K(b *testing.B) {
	benchmarkWrapper(4096, b)
}

func BenchmarkSha1Reader1M(b *testing.B) {
	benchmarkWrapper(1048576, b)
}
