// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

//go:build unix

package localsocket

import "net"

// PeerIdentity returns the credentials of the peer process using the
// platform-specific peerCred implementation.
func (c *unixConn) PeerIdentity() (PeerIdentity, error) {
	return peerIdentity(c.UnixConn)
}

// peerIdentity extracts peer credentials from a Unix connection.
func peerIdentity(c *net.UnixConn) (PeerIdentity, error) {
	raw, err := c.SyscallConn()
	if err != nil {
		return PeerIdentity{}, err
	}
	var id PeerIdentity
	var innerErr error
	err = raw.Control(func(fd uintptr) {
		id, innerErr = peerCred(int(fd))
	})
	if err != nil {
		return PeerIdentity{}, err
	}
	if innerErr != nil {
		return PeerIdentity{}, innerErr
	}
	return id, nil
}
