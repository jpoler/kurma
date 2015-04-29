// Copyright 2013 Apcera Inc. All rights reserved.

package testtool

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTestHttpGet(t *testing.T) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "GET" {
			w.WriteHeader(200)
			w.Write([]byte("Hello"))
		} else {
			w.WriteHeader(404)
			w.Write([]byte("Not Found"))
		}
	}

	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	m := &MockLogger{}

	// test success without checking code
	m.RunTest(t, false, func() {
		body, code := TestHttpGet(m, s.URL, -1)
		TestEqual(m, code, 200)
		TestEqual(m, body, "Hello")
	})

	// test checking code
	m.RunTest(t, false, func() {
		TestHttpGet(m, s.URL, 200)
	})

	// test checking code with incorrect value
	m.RunTest(t, true, func() {
		TestHttpGet(m, s.URL, 404)
	})
}

func TestTestHttpPost(t *testing.T) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "POST" {
			w.WriteHeader(200)
			w.Write([]byte("Hello"))
		} else {
			w.WriteHeader(404)
			w.Write([]byte("Not Found"))
		}
	}

	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	m := &MockLogger{}

	// test success without checking code
	m.RunTest(t, false, func() {
		body, code := TestHttpPost(m, s.URL, "", "", -1)
		TestEqual(m, code, 200)
		TestEqual(m, body, "Hello")
	})

	// test checking code
	m.RunTest(t, false, func() {
		TestHttpPost(m, s.URL, "", "", 200)
	})

	// test checking code with incorrect value
	m.RunTest(t, true, func() {
		TestHttpPost(m, s.URL, "", "", 404)
	})
}

func TestTestHttpPut(t *testing.T) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "PUT" {
			w.WriteHeader(200)
			w.Write([]byte("Hello"))
		} else {
			w.WriteHeader(404)
			w.Write([]byte("Not Found"))
		}
	}

	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	m := &MockLogger{}

	// test success without checking code
	m.RunTest(t, false, func() {
		body, code := TestHttpPut(m, s.URL, "", "", -1)
		TestEqual(m, code, 200)
		TestEqual(m, body, "Hello")
	})

	// test checking code
	m.RunTest(t, false, func() {
		TestHttpPut(m, s.URL, "", "", 200)
	})

	// test checking code with incorrect value
	m.RunTest(t, true, func() {
		TestHttpPut(m, s.URL, "", "", 404)
	})
}

func TestTestHttpPostAndPutSendBody(t *testing.T) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		io.Copy(w, req.Body)
	}

	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	m := &MockLogger{}

	// check blank body
	m.RunTest(t, false, func() {
		body, code := TestHttpPost(m, s.URL, "", "", -1)
		TestEqual(m, code, 200)
		TestEqual(m, body, "")
	})
	m.RunTest(t, false, func() {
		body, code := TestHttpPut(m, s.URL, "", "", -1)
		TestEqual(m, code, 200)
		TestEqual(m, body, "")
	})

	// check with a body
	m.RunTest(t, false, func() {
		body, code := TestHttpPost(m, s.URL, "text/plain", "sample", -1)
		TestEqual(m, code, 200)
		TestEqual(m, body, "sample")
	})
	m.RunTest(t, false, func() {
		body, code := TestHttpPut(m, s.URL, "text/plain", "sample", -1)
		TestEqual(m, code, 200)
		TestEqual(m, body, "sample")
	})
}

func TestTestHttpMethodsInvalidUrl(t *testing.T) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("Hello"))
	}

	s := httptest.NewServer(http.HandlerFunc(handler))
	s.Close()

	m := &MockLogger{}

	m.RunTest(t, true, func() { TestHttpGet(m, s.URL, -1) })
	m.RunTest(t, true, func() { TestHttpPost(m, s.URL, "", "", -1) })
	m.RunTest(t, true, func() { TestHttpPut(m, s.URL, "", "", -1) })
}
