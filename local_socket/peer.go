// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

package localsocket

// PeerIdentity reports the credentials of the process on the other end of a
// local socket connection. File permissions say who may connect; peer
// credentials say who did. A zero-valued field means the platform does not
// report it.
type PeerIdentity struct {
	// PID is the peer process ID, 0 if the platform does not report it.
	PID int

	// UID is the peer user ID (Unix only).
	UID uint32

	// GID is the peer group ID (Unix only).
	GID uint32

	// SID is the peer security identifier (Windows only).
	SID string
}
