// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

// Command echo demonstrates interprocess-go/local_socket with a simple
// request/echo loop: a listener echoes every byte it receives back to the
// sender.
//
//	go run ./examples/echo -mode server
//	go run ./examples/echo -mode client
package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"

	localsocket "github.com/four-bytes/interprocess-go/local_socket"
)

func main() {
	mode := flag.String("mode", "server", "server or client")
	name := flag.String("name", "echo", "socket identifier (UserScoped)")
	runtimeDir := flag.String("runtime-dir", "", "explicit runtime directory (server only)")
	flag.Parse()

	switch *mode {
	case "server":
		runServer(*name, *runtimeDir)
	case "client":
		runClient(*name)
	default:
		log.Fatalf("unknown mode %q (want server or client)", *mode)
	}
}

func runServer(name, runtimeDir string) {
	ln, err := localsocket.Listen(
		localsocket.UserScoped(name),
		localsocket.ListenOptions{RuntimeDir: runtimeDir, ReclaimStale: true},
	)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Printf("listening on %s", ln.Addr())

	for {
		c, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go func(c io.ReadWriteCloser) {
			defer c.Close()
			if _, err := io.Copy(c, c); err != nil {
				log.Printf("echo: %v", err)
			}
		}(c)
	}
}

func runClient(name string) {
	// The client resolves the name through the default runtime directory, so
	// both sides meet on the same endpoint. For a custom server runtime
	// directory, pass the exact path the server printed instead of a name.
	c, err := localsocket.Dial(context.Background(), localsocket.UserScoped(name), localsocket.DialOptions{})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	msg := []byte("hello over a local socket\n")
	if _, err := c.Write(msg); err != nil {
		log.Fatal(err)
	}
	// Read exactly the echoed bytes back. The server keeps the connection open
	// for further messages, so copying until EOF would block forever.
	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(c, buf); err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stdout.Write(buf); err != nil {
		log.Fatal(err)
	}
}
