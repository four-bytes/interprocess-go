// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

//go:build unix

package localsocket

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"syscall"
)

// runtimeDirCandidates returns the runtime-directory candidates in Decision 2
// precedence order. The $TMPDIR step is gated to Darwin; /run/user/$UID to
// Linux. No candidate is trusted: each is validated before use.
func runtimeDirCandidates(explicit string) []string {
	var dirs []string
	if explicit != "" {
		dirs = append(dirs, explicit)
	}
	if x := os.Getenv("XDG_RUNTIME_DIR"); x != "" {
		dirs = append(dirs, x)
	}
	if runtime.GOOS == "darwin" {
		if t := os.Getenv("TMPDIR"); t != "" {
			dirs = append(dirs, t)
		}
	}
	if runtime.GOOS == "linux" {
		dirs = append(dirs, fmt.Sprintf("/run/user/%d", os.Getuid()))
	}
	if c, err := os.UserCacheDir(); err == nil && c != "" {
		dirs = append(dirs, c)
	}
	return dirs
}

// resolveRuntimeDir returns the first candidate that passes validation, or
// ErrNoRuntimeDir. A failing candidate is skipped, never repaired.
func resolveRuntimeDir(explicit string) (string, error) {
	for _, c := range runtimeDirCandidates(explicit) {
		if err := validateRuntimeDir(c); err == nil {
			return c, nil
		}
	}
	return "", ErrNoRuntimeDir
}

// validateRuntimeDir enforces security invariant 2: the directory must exist,
// be a directory, be owned by the current UID, and have no group or world
// write bit.
func validateRuntimeDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot inspect ownership of %s", path)
	}
	if int(st.Uid) != os.Getuid() {
		return fmt.Errorf("%s is not owned by the current user", path)
	}
	if st.Mode&0o022 != 0 {
		return fmt.Errorf("%s has group or world write bits", path)
	}
	return nil
}

// ensurePrivateDir creates the library's private 0700 subdirectory under a
// validated runtime directory, or verifies and repairs an existing one owned
// by the current user.
func ensurePrivateDir(dir string) error {
	err := os.Mkdir(dir, 0o700)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("runtime subdirectory %s is not a directory", dir)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot inspect ownership of %s", dir)
	}
	if int(st.Uid) != os.Getuid() {
		return fmt.Errorf("runtime subdirectory %s is not owned by the current user", dir)
	}
	// Enforce 0700; we own the directory so the repair is safe.
	return os.Chmod(dir, 0o700)
}
