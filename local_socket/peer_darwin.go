// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

//go:build darwin

package localsocket

import "syscall"

// peerCred returns the peer's UID and GID via getpeereid. macOS does not
// report the peer PID.
func peerCred(fd int) (PeerIdentity, error) {
	uid, gid, err := syscall.Getpeereid(fd)
	if err != nil {
		return PeerIdentity{}, err
	}
	return PeerIdentity{UID: uint32(uid), GID: uint32(gid)}, nil
}
