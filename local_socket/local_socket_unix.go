// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

//go:build unix

package localsocket

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"syscall"
)

// Listen creates a listener for name and returns it as a net.Listener. The
// returned value also implements Listener (LocalSocketName). The runtime
// directory and socket are created with restrictive modes (0700 / 0600), and
// stale cleanup only ever removes an owned, provably-stale socket.
func Listen(name Name, options ListenOptions) (net.Listener, error) {
	path, err := resolveListenPath(name, options)
	if err != nil {
		return nil, err
	}

	if err := prepareSocketPath(path, options.ReclaimStale); err != nil {
		return nil, err
	}

	ln, err := net.Listen("unix", path)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			return nil, fmt.Errorf("%w: %v", ErrAlreadyInUse, err)
		}
		if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) {
			return nil, fmt.Errorf("%w: %v", ErrPermissionDenied, err)
		}
		return nil, err
	}

	// Go's UnixListener unlinks a socket it created when Close is called,
	// which is the behaviour we want by default. KeepOnClose opts out.
	ul := ln.(*net.UnixListener)
	ul.SetUnlinkOnClose(!options.KeepOnClose)

	// Step 6: set restrictive mode bits and keep them.
	if err := os.Chmod(path, 0o600); err != nil {
		ln.Close()
		os.Remove(path)
		return nil, err
	}

	return &unixListener{ln: ul, name: name}, nil
}

// resolveListenPath resolves name to a filesystem socket path, creating and
// validating the private runtime directory for non-filesystem names.
func resolveListenPath(name Name, options ListenOptions) (string, error) {
	if name.Kind == NameFilesystem {
		return name.Value, nil
	}
	if err := validateIdentifier(name.Value); err != nil {
		return "", err
	}
	base, err := resolveRuntimeDir(options.RuntimeDir)
	if err != nil {
		return "", err
	}
	dir := base + "/interprocess-go"
	if err := ensurePrivateDir(dir); err != nil {
		return "", err
	}
	return joinSocketPath(dir, maybeTruncate(name.Value, dir)), nil
}

// resolveDialPath resolves name for Dial: identical resolution to Listen but
// without creating anything (the listener owns directory creation).
func resolveDialPath(name Name) (string, error) {
	if name.Kind == NameFilesystem {
		return name.Value, nil
	}
	if err := validateIdentifier(name.Value); err != nil {
		return "", err
	}
	base, err := resolveRuntimeDir("")
	if err != nil {
		return "", err
	}
	dir := base + "/interprocess-go"
	return joinSocketPath(dir, maybeTruncate(name.Value, dir)), nil
}

// Dial connects to name, honouring context cancellation and Timeout. The
// returned value also implements Conn (PeerIdentity).
func Dial(ctx context.Context, name Name, options DialOptions) (net.Conn, error) {
	path, err := resolveDialPath(name)
	if err != nil {
		return nil, err
	}
	d := net.Dialer{Timeout: options.Timeout}
	c, err := d.DialContext(ctx, "unix", path)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) {
			return nil, fmt.Errorf("%w: %v", ErrPermissionDenied, err)
		}
		return nil, err
	}
	return &unixConn{UnixConn: c.(*net.UnixConn)}, nil
}

// unixListener wraps *net.UnixListener to add LocalSocketName and to make a
// closed listener report net.ErrClosed from Accept.
type unixListener struct {
	ln     *net.UnixListener
	name   Name
	closed atomic.Bool
}

func (l *unixListener) Accept() (net.Conn, error) {
	c, err := l.ln.AcceptUnix()
	if err != nil {
		if l.closed.Load() || errors.Is(err, net.ErrClosed) {
			return nil, net.ErrClosed
		}
		return nil, err
	}
	return &unixConn{UnixConn: c}, nil
}

func (l *unixListener) Close() error {
	l.closed.Store(true)
	return l.ln.Close()
}

func (l *unixListener) Addr() net.Addr {
	return l.ln.Addr()
}

func (l *unixListener) LocalSocketName() Name {
	return l.name
}

// unixConn wraps *net.UnixConn to add PeerIdentity.
type unixConn struct {
	*net.UnixConn
}
