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

### Technical Details
- Module path: `github.com/four-bytes/interprocess-go`
- Go 1.24 minimum; standard library only on Unix, `Microsoft/go-winio` on Windows
- Phase 1 (Unix core) implemented (#1); Windows (Phase 2), interop (Phase 3) and framing
  (Phase 4) remain.
