// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package assert

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Equal[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func Zero[T comparable](t *testing.T, got T) {
	t.Helper()
	var zero T
	if got != zero {
		t.Errorf("got: %v, want zero value", got)
	}
}

func NotZero[T comparable](t *testing.T, got T) {
	t.Helper()
	var zero T
	if got == zero {
		t.Errorf("got zero, want non zero value")
	}
}

func Err(t *testing.T, err error) {
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func NotErr(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func Contains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("%q does not contain %q", got, want)
	}
}

func ContainsAll(t *testing.T, got string, substrings ...string) {
	t.Helper()
	for _, want := range substrings {
		if !strings.Contains(got, want) {
			t.Errorf("output = %q, want substring %q", got, want)
		}
	}
}

func ErrIs(t *testing.T, got, target error) {
	t.Helper()
	if !errors.Is(got, target) {
		t.Fatalf("err = %v, want %v", got, target)
	}
}

func ErrAs[T error](t *testing.T, got error) {
	t.Helper()
	var target T
	if !errors.As(got, &target) {
		t.Fatalf("err type = %T, want %T", got, target)
	}
}

func Nil(t *testing.T, got any) {
	t.Helper()
	if !isNil(got) {
		t.Errorf("got: %v, want nil", got)
	}
}

func NotNil(t *testing.T, got any) {
	t.Helper()
	if isNil(got) {
		t.Fatalf("expected non-nil, got nil")
	}
}

func NoDiff(t *testing.T, want, got any, opts ...cmp.Option) {
	t.Helper()
	if diff := cmp.Diff(want, got, opts...); diff != "" {
		t.Errorf("(-want, +got):\n%s", diff)
	}
}

func HasKey[K comparable, V any](t *testing.T, m map[K]V, key K) {
	t.Helper()
	if _, ok := m[key]; !ok {
		t.Errorf("map missing key %v", key)
	}
}

func NoKey[K comparable, V any](t *testing.T, m map[K]V, key K) {
	t.Helper()
	if _, ok := m[key]; ok {
		t.Errorf("map should not have key %v", key)
	}
}

func PathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("path %q should exist", path)
	}
}

func NoPathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("path %q should not exist", path)
	}
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}
