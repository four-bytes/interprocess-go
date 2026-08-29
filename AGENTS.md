# AGENTS.md

This file provides guidance to AI coding agents working in this repository.

## What is interprocess-go?

A small Go library providing local, connection-oriented byte streams over a unified API — Unix
domain sockets on Linux/macOS, named pipes on Windows. It is the deliberate counterpart to the
Rust `interprocess` crate, so a Rust host and a Go agent can share one endpoint strategy.

It abstracts the **local transport only**. No JSON-RPC, no event system, no persistence, no
service registry, no process management, no discovery — those belong to the application.

## Build & Development Commands

| Command | Description |
|---------|-------------|
| `go build ./...` | Build all packages |
| `go test -race ./...` | Run the test suite with the race detector |
| `go test -run TestVector ./...` | Run conformance vectors only |
| `go test -fuzz FuzzReadFrame ./framing` | Fuzz the frame reader |
| `go vet ./...` | Static analysis |
| `gofmt -l .` | List unformatted files (must be empty) |

## Architecture

- **`local_socket/`** — the public API: `Name`, `Listen`, `Dial`, options, security, peer identity
- **`framing/`** — optional 4-byte big-endian length prefix over any `io.Reader`/`io.Writer`
- **`internal/platform/`** — per-OS resolution and permission enforcement
- **`internal/testutil/`** — shared helpers for platform tests
- **`examples/`** — `echo`, `request_reply`, `rust_interop`
- **`testdata/`** — conformance vectors V1-V4, committed and asserted from both languages

## Conventions

- LF endings, UTF-8, tabs in Go, `gofmt` clean
- Build tags `//go:build unix` and `//go:build windows` for platform files
- Sentinel errors, inspectable via `errors.Is` / `errors.As`; never swallow a security-relevant cause
- SPDX header in every source file: `// SPDX-License-Identifier: Apache-2.0`
- Conventional commits, Issue → Branch → PR (see `CONTRIBUTE.md`)
- No personal paths in committed files; local notes go in `*.local.md` (gitignored)

## Security Invariants

These are not preferences. A change that breaks one is a defect, not a trade-off.

1. **No TCP fallback**, silent or otherwise. A local endpoint stays local.
2. **Every runtime-directory candidate is validated** — exists, is a directory, owned by the
   current UID, no group or world write bit — including one passed explicitly. None valid →
   `ErrNoRuntimeDir`, never a weaker fallback.
3. **`$TMPDIR` is a Darwin-only precedence step.** On Linux it conventionally means
   world-writable `/tmp`.
4. **Stale cleanup removes only an owned socket.** A regular file at the socket path is
   `ErrStaleCleanupUnsafe`, never an unlink.
5. **No Linux abstract namespace** — no filesystem permissions, so any process in the namespace
   could connect.
6. **`ReadFrame` checks the declared length before allocating.** Otherwise it is a remote
   memory-exhaustion primitive.
7. **Windows DACL grants the intended SID only** — never `Everyone`, `Anonymous`, or the
   default DACL.

## Tech Stack

- Go 1.24+, `CGO_ENABLED=0`
- Standard library only on Unix; `github.com/Microsoft/go-winio` on Windows
- Apache-2.0

## Design Source

The full design, decisions with rationale, conformance vectors and per-phase acceptance criteria
live in the `four-file-cloud` repository, `docs/interprocess-go-concept.md`. Mirrored here as
`docs/ARCHITECTURE.md`, `docs/INTEROP.md` and `docs/TESTING.md`.
