// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

//go:build darwin

package localsocket

import "golang.org/x/sys/unix"

// peerCred returns the peer's UID and primary GID.
//
// macOS has no SO_PEERCRED. The equivalent is getsockopt(SOL_LOCAL,
// LOCAL_PEERCRED), which fills a struct xucred. Note what it does and does not
// carry: a UID and a group list, but no PID -- so PeerIdentity.PID stays zero
// here, as documented. LOCAL_PEERPID exists for the PID but is a second call
// against a different option, and the handshake does not need it.
//
// There is no syscall.Getpeereid in the standard library; x/sys is the
// canonical source for this primitive, which is why this package takes that
// one dependency on Darwin.
func peerCred(fd int) (PeerIdentity, error) {
	xu, err := unix.GetsockoptXucred(fd, unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	if err != nil {
		return PeerIdentity{}, err
	}
	id := PeerIdentity{UID: xu.Uid}
	if xu.Ngroups > 0 {
		id.GID = xu.Groups[0]
	}
	return id, nil
}
