// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package testutil provides test assertion helpers and file utilities.
package testutil

import (
	"errors"
	"testing"
)

// Must wraps a [testing.T] to provide concise fatal assertions.
type Must struct {
	t *testing.T
}

// NewMust creates a Must-wrapper bound to t.
func NewMust(t *testing.T) Must {
	return Must{t: t}
}

// Err fails the test if err is nil.
func (m Must) Err(err error) {
	m.t.Helper()
	if err == nil {
		m.t.Fatal("expected error, got nil")
	}
}

// NotErr fails the test if err is non-nil.
func (m Must) NotErr(err error) {
	m.t.Helper()
	if err != nil {
		m.t.Fatal(err)
	}
}

// WantErr fails if got does not match target via [errors.Is].
func WantErr(t *testing.T, got, target error) {
	t.Helper()
	if !errors.Is(got, target) {
		t.Fatalf("err = %v, want %v", got, target)
	}
}

// WantErrAs fails if got cannot be unwrapped to type T via [errors.As].
func WantErrAs[T error](t *testing.T, got error) {
	t.Helper()
	var target T
	if !errors.As(got, &target) {
		t.Fatalf("err type = %T, want %T", got, target)
	}
}
