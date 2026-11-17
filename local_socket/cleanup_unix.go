// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

//go:build unix

package localsocket

import (
	"errors"
	"io/fs"
	"net"
	"os"
	"syscall"
	"time"
)

// staleDialTimeout bounds the liveness probe when reclaiming a stale socket.
const staleDialTimeout = 200 * time.Millisecond

// prepareSocketPath implements steps 3 and 4 of the documented six-step
// cleanup: if an entry exists at the socket path, validate its type and owner,
// then remove only a socket file that is owned by us and provably stale. A
// regular file, directory, symlink or foreign-owned socket is never removed.
func prepareSocketPath(path string, reclaimStale bool) error {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil // step 3: nothing there, bind directly
	}
	if err != nil {
		return err
	}

	// Step 3: validate file type and owner. Never follow a symlink.
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || info.Mode()&os.ModeSocket == 0 || int(st.Uid) != os.Getuid() {
		return ErrStaleCleanupUnsafe
	}

	// Step 4: an owned socket. Reclaim only if requested; the liveness probe
	// distinguishes a live listener from a stale one.
	if !reclaimStale {
		return ErrAlreadyInUse
	}
	conn, err := net.DialTimeout("unix", path, staleDialTimeout)
	if err == nil {
		conn.Close()
		return ErrAlreadyInUse
	}
	return os.Remove(path)
}
