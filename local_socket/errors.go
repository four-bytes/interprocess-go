// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

// Package localsocket provides local, connection-oriented byte streams over a
// unified API: Unix domain sockets on Linux and macOS, named pipes on Windows.
// It abstracts the local transport only; framing, discovery and process
// management belong to the application.
package localsocket

import "errors"

// Sentinel errors. Every one is inspectable via errors.Is; the library never
// swallows a security-relevant underlying error.
var (
	// ErrInvalidName reports a name that failed validation (empty, wrong
	// character set, or containing path separators).
	ErrInvalidName = errors.New("invalid local socket name")

	// ErrUnsupportedName reports a name kind that the current platform does
	// not support (for example, a filesystem path on Windows).
	ErrUnsupportedName = errors.New("unsupported name type on this platform")

	// ErrPermissionDenied reports that the endpoint could not be reached or
	// bound because of a permission failure.
	ErrPermissionDenied = errors.New("local socket permission denied")

	// ErrAlreadyInUse reports that the endpoint already has a live listener
	// (or an entry that Listen is not permitted to reclaim).
	ErrAlreadyInUse = errors.New("local socket already in use")

	// ErrStaleCleanupUnsafe reports that an entry exists at the socket path
	// which stale cleanup refuses to remove: a regular file, a directory, a
	// symbolic link, or a socket not owned by the current UID.
	ErrStaleCleanupUnsafe = errors.New("refusing unsafe stale socket cleanup")

	// ErrNoRuntimeDir reports that no runtime-directory candidate passed
	// ownership and mode validation. Nothing is created in this case.
	ErrNoRuntimeDir = errors.New("no runtime directory passed ownership and mode validation")

	// ErrPeerIdentityUnsupported reports that peer credentials are not
	// available on this platform. Callers must handle it rather than reading
	// the zero value of PeerIdentity as "nobody".
	ErrPeerIdentityUnsupported = errors.New("peer identity unavailable on this platform")
)
