// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

package localsocket

import (
	"strings"
	"testing"
)

func TestValidateIdentifier(t *testing.T) {
	valid := []string{"file-core-agent", "file-core-agent-01", "a.b_c-d"}
	for _, id := range valid {
		if err := validateIdentifier(id); err != nil {
			t.Errorf("validateIdentifier(%q) = %v, want nil", id, err)
		}
	}

	invalid := []string{"", "has/slash", `has\backslash`, "has space", "ümlaut", "has:colon", "has+plus"}
	for _, id := range invalid {
		if err := validateIdentifier(id); !isErr(err, ErrInvalidName) {
			t.Errorf("validateIdentifier(%q) = %v, want ErrInvalidName", id, err)
		}
	}
}

func TestConstructors(t *testing.T) {
	if got := Filesystem("/tmp/x.sock"); got.Kind != NameFilesystem || got.Value != "/tmp/x.sock" {
		t.Errorf("Filesystem = %+v", got)
	}
	if got := Namespaced("agent"); got.Kind != NameNamespaced || got.Value != "agent" {
		t.Errorf("Namespaced = %+v", got)
	}
	if got := UserScoped("agent"); got.Kind != NameNamespaced || got.Value != "agent" {
		t.Errorf("UserScoped = %+v", got)
	}
}

// TestTruncateIdentifierVector pins the Decision 6 output for the V3
// 200-character case.
func TestTruncateIdentifierVector(t *testing.T) {
	id := strings.Repeat("a", 200)
	got := truncateIdentifier(id)
	want := "aaaaaaaaaaaaaaaa-c2a908d9"
	if got != want {
		t.Fatalf("truncateIdentifier(200 x 'a') = %q, want %q", got, want)
	}
	if len(got) != 25 {
		t.Fatalf("truncated length = %d, want 25", len(got))
	}
}

func TestMaybeTruncate(t *testing.T) {
	// A short identifier under a short directory is unchanged.
	if got := maybeTruncate("agent", "/tmp/x"); got != "agent" {
		t.Fatalf("maybeTruncate short = %q, want agent", got)
	}
	// A 200-char identifier always exceeds the platform limit and truncates.
	long := strings.Repeat("a", 200)
	if got := maybeTruncate(long, "/tmp/x"); got != "aaaaaaaaaaaaaaaa-c2a908d9" {
		t.Fatalf("maybeTruncate long = %q, want truncated", got)
	}
}

func isErr(err, target error) bool {
	return err == target
}
