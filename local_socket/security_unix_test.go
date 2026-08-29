// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

//go:build unix

package localsocket

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateRuntimeDir(t *testing.T) {
	good := t.TempDir() // 0700, owned by us
	if err := validateRuntimeDir(good); err != nil {
		t.Fatalf("0700 owned dir must validate: %v", err)
	}

	worldWritable := t.TempDir()
	if err := os.Chmod(worldWritable, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := validateRuntimeDir(worldWritable); err == nil {
		t.Fatal("0777 dir must be rejected")
	}

	groupWritable := t.TempDir()
	if err := os.Chmod(groupWritable, 0o770); err != nil {
		t.Fatal(err)
	}
	if err := validateRuntimeDir(groupWritable); err == nil {
		t.Fatal("group-writable dir must be rejected")
	}

	notDir := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(notDir, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateRuntimeDir(notDir); err == nil {
		t.Fatal("regular file must be rejected")
	}

	missing := filepath.Join(t.TempDir(), "nope")
	if err := validateRuntimeDir(missing); err == nil {
		t.Fatal("missing dir must be rejected")
	}
}

// TestExplicitRuntimeDirValidated proves that even an explicitly passed
// RuntimeDir is subject to owner/mode validation (invariant 2).
func TestExplicitRuntimeDirValidated(t *testing.T) {
	bad := t.TempDir()
	if err := os.Chmod(bad, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := validateRuntimeDir(bad); err == nil {
		t.Fatal("explicit 0777 runtime dir must be rejected")
	}
}

// TestTMPDIRIgnoredOnLinux pins the Darwin-only precedence step for $TMPDIR.
func TestTMPDIRIgnoredOnLinux(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("$TMPDIR is a legitimate candidate on darwin")
	}
	d := t.TempDir()
	if err := os.Chmod(d, 0o777); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", d)
	for _, c := range runtimeDirCandidates("") {
		if c == d {
			t.Fatal("$TMPDIR must not be a runtime-dir candidate on Linux")
		}
	}
}

// TestNoValidCandidateReturnsErrNoRuntimeDir covers criterion 1.4: with no
// valid candidate, resolution and Listen fail with ErrNoRuntimeDir and nothing
// is created.
func TestNoValidCandidateReturnsErrNoRuntimeDir(t *testing.T) {
	bad := t.TempDir()
	if err := os.Chmod(bad, 0o777); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_RUNTIME_DIR", bad)
	t.Setenv("XDG_CACHE_HOME", bad)
	t.Setenv("HOME", bad)

	runUser := fmt.Sprintf("/run/user/%d", os.Getuid())
	if _, err := os.Stat(runUser); err == nil {
		t.Skipf("%s exists in this environment; cannot force ErrNoRuntimeDir", runUser)
	}

	if _, err := resolveRuntimeDir(""); !errors.Is(err, ErrNoRuntimeDir) {
		t.Fatalf("resolveRuntimeDir = %v, want ErrNoRuntimeDir", err)
	}

	_, err := Listen(UserScoped("nocandidate"), ListenOptions{})
	if !errors.Is(err, ErrNoRuntimeDir) {
		t.Fatalf("Listen = %v, want ErrNoRuntimeDir", err)
	}
	if _, err := os.Stat(filepath.Join(bad, "interprocess-go")); !os.IsNotExist(err) {
		t.Fatal("Listen must not create anything inside a rejected candidate")
	}
}
