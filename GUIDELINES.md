# Coding Guidelines — interprocess-go

## Tech Stack

- Runtime: Go 1.25+, `CGO_ENABLED=0` for builds
- Language: Go
- Dependencies: standard library only on Unix; `github.com/Microsoft/go-winio` on Windows
- License: Apache-2.0

## Code Style

- `gofmt` is the formatter; `gofmt -l .` must print nothing before commit
- Tabs for indentation, LF endings, UTF-8
- Platform files use build tags — `//go:build unix` and `//go:build windows`
- Exported identifiers carry doc comments beginning with the identifier name
- Errors are wrapped with `%w`; sentinels are inspectable via `errors.Is` / `errors.As`
- Never swallow a security-relevant underlying error

## License Header

Every source file starts with:

```go
// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes
```

## File Conventions

- LF line endings, UTF-8 encoding
- `*.local.md` is gitignored — use it for personal notes
- No personal paths in committed files

## Security Invariants

A change that breaks one of these is a defect, not a trade-off. Each has a test that fails if it
is violated; if you find one without a test, that is a bug worth its own issue.

1. **No TCP fallback**, silent or otherwise.
2. **Every runtime-directory candidate is validated** — exists, is a directory, owned by the
   current UID, no group or world write bit. This applies to an explicitly passed `RuntimeDir`
   too. If none passes, fail with `ErrNoRuntimeDir`.
3. **`$TMPDIR` is a Darwin-only precedence step.** On macOS it is per-user and mode `0700`; on
   Linux it conventionally means world-writable `/tmp`, which must never hold a socket.
4. **Stale cleanup removes only an owned socket.** Anything else at the socket path —
   a regular file, a directory, another user's socket — is `ErrStaleCleanupUnsafe`.
5. **No Linux abstract namespace.** Abstract sockets carry no filesystem permissions, so every
   process in the network namespace could connect. This is declined permanently, not deferred.
6. **`ReadFrame` validates the declared length before allocating.** A reader that allocates first
   is a remote memory-exhaustion primitive.
7. **The Windows DACL grants the intended SID only** — never `Everyone`, never `Anonymous`,
   never the default DACL.

## Build Discipline

- Every change ends with `gofmt`, `go vet`, `go test -race ./...`
- Version bump and `HISTORY.md` entry before merge
- No merge with a red gate; fix or revert, never skip
