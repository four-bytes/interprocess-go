// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

package localsocket

import "net"

// Listener is the interface returned by Listen. It embeds net.Listener and adds
// the name the listener was created with. Every value returned by Listen
// satisfies it.
type Listener interface {
	net.Listener

	// LocalSocketName returns the Name the listener was created with.
	LocalSocketName() Name
}
