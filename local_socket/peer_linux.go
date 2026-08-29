// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

//go:build linux

package localsocket

import "syscall"

// peerCred returns the peer's PID, UID and GID via SO_PEERCRED.
func peerCred(fd int) (PeerIdentity, error) {
	cred, err := syscall.GetsockoptUcred(fd, syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if err != nil {
		return PeerIdentity{}, err
	}
	return PeerIdentity{PID: int(cred.Pid), UID: cred.Uid, GID: cred.Gid}, nil
}
