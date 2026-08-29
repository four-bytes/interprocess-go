// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

package localsocket

import "net"

// Conn is the interface returned by Dial and Accept. It embeds net.Conn and
// adds peer identity. Every connection produced by this package satisfies it.
type Conn interface {
	net.Conn

	// PeerIdentity returns the credentials of the peer process, or
	// ErrPeerIdentityUnsupported where the platform cannot supply them.
	PeerIdentity() (PeerIdentity, error)
}
