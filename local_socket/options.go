// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

package localsocket

import "time"

// AccessPolicy selects who may reach a local endpoint.
type AccessPolicy uint8

const (
	// AccessCurrentUser restricts the endpoint to the current user. It is the
	// zero value and therefore the secure default: a caller passing an empty
	// ListenOptions gets it without any opt-in. On Unix this is a mode-0600
	// socket under a mode-0700 runtime directory.
	AccessCurrentUser AccessPolicy = iota

	// AccessCurrentLogonSession additionally scopes to the current logon
	// session (Windows only; on Unix it is equivalent to AccessCurrentUser).
	AccessCurrentLogonSession

	// AccessCustom defers to an explicit platform security descriptor
	// (Windows only; on Unix it is equivalent to AccessCurrentUser).
	AccessCustom
)

// ListenOptions configures a Listen call.
type ListenOptions struct {
	// Access selects the endpoint access policy. AccessCurrentUser (the zero
	// value) is the secure default.
	Access AccessPolicy

	// RuntimeDir overrides runtime-directory resolution with an explicit base
	// directory. It is validated exactly like every implicit candidate: it
	// must exist, be a directory, be owned by the current UID, and have no
	// group or world write bit. A value that fails is skipped, not repaired;
	// if nothing else survives, Listen returns ErrNoRuntimeDir.
	RuntimeDir string

	// ReclaimStale, when true, tells Listen to remove a stale socket left
	// behind by a previous, no-longer-running listener. Reclamation is always
	// safe: only a socket file owned by the current UID and provably stale is
	// removed. A regular file, directory, symlink or foreign-owned socket at
	// the endpoint always yields ErrStaleCleanupUnsafe, and a live socket
	// always yields ErrAlreadyInUse, whether or not ReclaimStale is set.
	ReclaimStale bool

	// RemoveOnClose, when true, removes the listener's own socket file when
	// Close is called, releasing the name for the next listener. The zero
	// value leaves the file behind; a later Listen must then reclaim it via
	// ReclaimStale or fail with ErrAlreadyInUse.
	RemoveOnClose bool
}

// DialOptions configures a Dial call.
type DialOptions struct {
	// Timeout bounds the connect. A zero value means no timeout; cancellation
	// still applies through the context.
	Timeout time.Duration
}
