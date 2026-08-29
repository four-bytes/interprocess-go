# interprocess-go — Change History

## [Unreleased]

### Added
- Repository skeleton: Apache-2.0 license, README, CLAUDE.md, CONTRIBUTE.md, GUIDELINES.md,
  ROADMAP.md, `docs/` set, `.github/` templates, CI matrix over Linux/macOS/Windows.
- Design carried over from `four-file-cloud/docs/interprocess-go-concept.md` at D5
  (Implementable): six closed API decisions, conformance vectors V1-V4, per-phase acceptance
  criteria.
- Phase 1 Unix core: `local_socket` package (`Name`, `NameKind`, `Filesystem`, `Namespaced`,
  `UserScoped`, `Listen`, `Dial`, `ListenOptions`, `DialOptions`, `AccessPolicy`,
  `PeerIdentity`, error sentinels) — Unix domain sockets on Linux and macOS, Decision 2
  runtime-directory resolution with owner/mode validation, the safe six-step stale-socket
  cleanup, and peer identity via `SO_PEERCRED` (Linux) / `getpeereid` (macOS) (#1).
- `examples/echo` demonstrating a local echo server and client (#1).

### Changed
- `ListenOptions.RemoveOnClose` replaced by `KeepOnClose` with inverted meaning (#1). The zero
  value now releases the socket on `Close`, so a service restarts on default settings; the old
  opt-in meant every caller had to set two options or hit `ErrAlreadyInUse` on restart. It also
  matches Go (`net.UnixListener` unlinks a socket it created) and Rust `interprocess`, where name
  release is part of local-socket semantics. `ReclaimStale` stays opt-in, deliberately: remove
  what this process created, never touch what it did not.

### Fixed
- Criterion 1.4 (`ErrNoRuntimeDir`) was unreachable: the test skipped itself whenever
  `/run/user/$UID` existed, which is every systemd host (#1). The candidate chain is now injectable
  and the test covers both an empty set and a set where every candidate fails validation, plus the
  assertion that nothing is written into a rejected directory.

### Technical Details
- Module path: `github.com/four-bytes/interprocess-go`
- Go 1.24 minimum; standard library only on Unix, `Microsoft/go-winio` on Windows
- Phase 1 (Unix core) implemented (#1); Windows (Phase 2), interop (Phase 3) and framing
  (Phase 4) remain.
