/*
gophish - Open-Source Phishing Framework

The MIT License (MIT)

Copyright (c) 2013 Jordan Wright

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package context

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetReturnsNilForMissingKey(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	val := Get(r, "missing_key")
	if val != nil {
		t.Fatalf("expected nil for missing key, got %v", val)
	}
}

func TestSetAndGet(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = Set(r, "user_id", int64(42))
	if r == nil {
		t.Fatal("expected non-nil request from Set")
	}
	val := Get(r, "user_id")
	if val == nil {
		t.Fatal("expected non-nil value from Get")
	}
	id, ok := val.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T", val)
	}
	if id != 42 {
		t.Fatalf("expected 42, got %d", id)
	}
}

func TestSetNilValueReturnsSameRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r2 := Set(r, "key", nil)
	// When val is nil, Set should return the same request unchanged
	if r2 != r {
		t.Fatal("expected same request object when setting nil value")
	}
}

func TestSetMultipleKeys(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = Set(r, "key1", "value1")
	r = Set(r, "key2", 123)

	val1 := Get(r, "key1")
	if val1 != "value1" {
		t.Fatalf("expected 'value1', got %v", val1)
	}
	val2 := Get(r, "key2")
	if val2 != 123 {
		t.Fatalf("expected 123, got %v", val2)
	}
}

func TestClearIsNoOp(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = Set(r, "key", "value")
	// Clear is a no-op in Go 1.7+, should not panic
	Clear(r)
	// Value should still be accessible
	val := Get(r, "key")
	if val != "value" {
		t.Fatalf("expected 'value' after Clear, got %v", val)
	}
}
