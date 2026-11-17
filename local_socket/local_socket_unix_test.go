// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

//go:build unix

package localsocket

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// echoServer starts an echo listener and returns a stop function. Each accepted
// connection is echoed back until the client closes.
func echoServer(t *testing.T, ln net.Listener) {
	t.Helper()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return // closed
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}(c)
		}
	}()
}

func TestEchoUserScoped(t *testing.T) {
	ln, err := Listen(UserScoped("echo"), ListenOptions{ReclaimStale: true})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if l, ok := ln.(Listener); !ok || l.LocalSocketName() != UserScoped("echo") {
		t.Fatal("Listen must return a Listener reporting its name")
	}
	echoServer(t, ln)

	c, err := Dial(context.Background(), UserScoped("echo"), DialOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, ok := c.(Conn); !ok {
		t.Fatal("Dial must return a Conn")
	}

	msg := []byte("hello over a local socket")
	if _, err := c.Write(msg); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(msg))
	if _, err := io.ReadFull(c, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, msg) {
		t.Fatalf("echo = %q, want %q", got, msg)
	}
}

func TestEchoNamespaced(t *testing.T) {
	ln, err := Listen(Namespaced("nsecho"), ListenOptions{ReclaimStale: true})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	echoServer(t, ln)

	c, err := Dial(context.Background(), Namespaced("nsecho"), DialOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Write([]byte("ns")); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, 2)
	if _, err := io.ReadFull(c, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != "ns" {
		t.Fatalf("echo = %q", got)
	}
}

func TestEchoFilesystem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fs.sock")
	ln, err := Listen(Filesystem(path), ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	echoServer(t, ln)

	c, err := Dial(context.Background(), Filesystem(path), DialOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Write([]byte("fs")); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, 2)
	if _, err := io.ReadFull(c, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != "fs" {
		t.Fatalf("echo = %q", got)
	}
}

func TestRestrictiveModes(t *testing.T) {
	base := t.TempDir()
	ln, err := Listen(UserScoped("modes"), ListenOptions{RuntimeDir: base})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	sub := filepath.Join(base, "interprocess-go")
	di, err := os.Stat(sub)
	if err != nil {
		t.Fatal(err)
	}
	if di.Mode().Perm() != 0o700 {
		t.Fatalf("runtime dir mode = %o, want 700", di.Mode().Perm())
	}
	si, err := os.Stat(filepath.Join(sub, "modes.sock"))
	if err != nil {
		t.Fatal(err)
	}
	if si.Mode().Perm() != 0o600 {
		t.Fatalf("socket mode = %o, want 600", si.Mode().Perm())
	}
}

func TestStaleCleanupRefusesRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.sock")
	if err := os.WriteFile(path, []byte("not a socket"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Listen(Filesystem(path), ListenOptions{ReclaimStale: true})
	if !errors.Is(err, ErrStaleCleanupUnsafe) {
		t.Fatalf("Listen = %v, want ErrStaleCleanupUnsafe", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("regular file must not be removed: %v", err)
	}
}

func TestStaleCleanupRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "link.sock")
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	_, err := Listen(Filesystem(path), ListenOptions{ReclaimStale: true})
	if !errors.Is(err, ErrStaleCleanupUnsafe) {
		t.Fatalf("Listen = %v, want ErrStaleCleanupUnsafe", err)
	}
}

func TestRestartAfterCrash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "crash.sock")

	// Simulate a crashed listener: bind, then close without unlinking.
	// net.Listen("unix", ...) unlinks on Close, so use ListenUnix with
	// unlink-on-close disabled — the socket file then remains behind exactly
	// as after a SIGKILL.
	laddr := &net.UnixAddr{Name: path, Net: "unix"}
	raw, err := net.ListenUnix("unix", laddr)
	if err != nil {
		t.Fatal(err)
	}
	raw.SetUnlinkOnClose(false)
	raw.Close()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("socket file should remain after crash: %v", err)
	}

	ln, err := Listen(Filesystem(path), ListenOptions{ReclaimStale: true})
	if err != nil {
		t.Fatalf("restart after crash failed: %v", err)
	}
	ln.Close()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("socket file should be removed on close, err=%v", err)
	}
}

func TestAlreadyInUse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "busy.sock")
	ln1, err := Listen(Filesystem(path), ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer ln1.Close()

	_, err = Listen(Filesystem(path), ListenOptions{ReclaimStale: true})
	if !errors.Is(err, ErrAlreadyInUse) {
		t.Fatalf("second Listen = %v, want ErrAlreadyInUse", err)
	}
}

func TestCloseUnblocksAccept(t *testing.T) {
	path := filepath.Join(t.TempDir(), "close.sock")
	ln, err := Listen(Filesystem(path), ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error, 1)
	go func() {
		_, err := ln.Accept()
		errCh <- err
	}()
	time.Sleep(50 * time.Millisecond)
	ln.Close()

	select {
	case err := <-errCh:
		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("Accept after Close = %v, want net.ErrClosed", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Accept did not unblock on Close")
	}
}

func TestCloseRemovesSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "remove.sock")
	ln1, err := Listen(Filesystem(path), ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := ln1.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("socket file should be removed on Close, err=%v", err)
	}

	// A second Listen on the same name now succeeds without reclaim.
	ln2, err := Listen(Filesystem(path), ListenOptions{})
	if err != nil {
		t.Fatalf("second Listen failed: %v", err)
	}
	ln2.Close()
}

func TestDialContextCancel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cancel.sock")
	before := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	_, err := Dial(ctx, Filesystem(path), DialOptions{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Dial = %v, want context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("cancelled dial not prompt: %v", elapsed)
	}

	time.Sleep(100 * time.Millisecond)
	if after := runtime.NumGoroutine(); after > before+2 {
		t.Fatalf("possible goroutine leak: before=%d after=%d", before, after)
	}
}

func TestDialContextDeadline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deadline.sock")
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Millisecond))
	defer cancel()
	time.Sleep(20 * time.Millisecond)
	_, err := Dial(ctx, Filesystem(path), DialOptions{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Dial = %v, want context.DeadlineExceeded", err)
	}
}

func TestDialTimeoutOption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "timeout.sock")
	ln, err := Listen(Filesystem(path), ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	c, err := Dial(context.Background(), Filesystem(path), DialOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("dial with generous timeout: %v", err)
	}
	c.Close()
}

func TestPeerIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "peer.sock")
	ln, err := Listen(Filesystem(path), ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		accepted <- c
	}()

	c, err := Dial(context.Background(), Filesystem(path), DialOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	id, err := c.(Conn).PeerIdentity()
	if err != nil {
		t.Fatalf("PeerIdentity: %v", err)
	}
	if id.UID != uint32(os.Getuid()) {
		t.Fatalf("UID = %d, want %d", id.UID, os.Getuid())
	}
	if id.GID != uint32(os.Getgid()) {
		t.Fatalf("GID = %d, want %d", id.GID, os.Getgid())
	}
	switch runtime.GOOS {
	case "linux":
		if id.PID != os.Getpid() {
			t.Fatalf("PID = %d, want %d", id.PID, os.Getpid())
		}
	case "darwin":
		if id.PID != 0 {
			t.Fatalf("darwin must not report PID, got %d", id.PID)
		}
	}

	srv := <-accepted
	defer srv.Close()
	sid, err := srv.(Conn).PeerIdentity()
	if err != nil {
		t.Fatalf("server PeerIdentity: %v", err)
	}
	if sid.UID != uint32(os.Getuid()) {
		t.Fatalf("server UID = %d, want %d", sid.UID, os.Getuid())
	}
}

func TestConcurrentEcho64x1MiB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "echo.sock")
	ln, err := Listen(Filesystem(path), ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				data, err := io.ReadAll(c)
				if err != nil {
					return
				}
				_, _ = c.Write(data)
			}(c)
		}
	}()

	const clients = 64
	const size = 1 << 20
	var wg sync.WaitGroup
	errCh := make(chan error, clients)
	wg.Add(clients)
	for i := 0; i < clients; i++ {
		go func(i int) {
			defer wg.Done()
			c, err := Dial(context.Background(), Filesystem(path), DialOptions{})
			if err != nil {
				errCh <- err
				return
			}
			defer c.Close()

			payload := make([]byte, size)
			for j := range payload {
				payload[j] = byte(i*31 + j)
			}
			if _, err := c.Write(payload); err != nil {
				errCh <- err
				return
			}
			// Half-close the write side so the server's ReadAll sees EOF; this
			// lets the echo complete without a write/read deadlock.
			if cw, ok := c.(interface{ CloseWrite() error }); ok {
				if err := cw.CloseWrite(); err != nil {
					errCh <- err
					return
				}
			}
			got := make([]byte, size)
			if _, err := io.ReadFull(c, got); err != nil {
				errCh <- err
				return
			}
			if !bytes.Equal(got, payload) {
				errCh <- fmt.Errorf("client %d: echoed data corrupted", i)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}

	ln.Close()
	<-serverDone
}

// TestCloseRemovesSocketByDefault covers criterion 1.7: Close releases the
// listener's own socket file and a second Listen on the same name succeeds --
// with no options set. A service that cannot restart on default settings is a
// defect, not a conservative default.
func TestCloseRemovesSocketByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "restart.sock")
	ln, err := Listen(Filesystem(path), ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("socket file must be gone after Close by default, err=%v", err)
	}

	ln2, err := Listen(Filesystem(path), ListenOptions{})
	if err != nil {
		t.Fatalf("second Listen after a clean Close must succeed without options: %v", err)
	}
	ln2.Close()
}

// TestKeepOnCloseLeavesSocket covers the opt-out: KeepOnClose leaves the file
// for a supervisor or socket-activation setup that owns the endpoint.
func TestKeepOnCloseLeavesSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keep.sock")
	ln, err := Listen(Filesystem(path), ListenOptions{KeepOnClose: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("socket file must remain when KeepOnClose is set: %v", err)
	}
	if _, err := Listen(Filesystem(path), ListenOptions{}); !errors.Is(err, ErrAlreadyInUse) {
		t.Fatalf("want ErrAlreadyInUse without ReclaimStale, got %v", err)
	}
	ln2, err := Listen(Filesystem(path), ListenOptions{ReclaimStale: true})
	if err != nil {
		t.Fatalf("reclaim of a kept socket failed: %v", err)
	}
	ln2.Close()
}

func TestLongNameResolution(t *testing.T) {
	base := t.TempDir()
	long := strings.Repeat("a", 200)
	ln, err := Listen(UserScoped(long), ListenOptions{RuntimeDir: base})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	want := filepath.Join(base, "interprocess-go", "aaaaaaaaaaaaaaaa-c2a908d9.sock")
	if got := ln.Addr().String(); got != want {
		t.Fatalf("resolved path = %q, want %q", got, want)
	}
}
