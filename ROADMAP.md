# interprocess-go — ROADMAP

**Owner:** Robby Beyer | **Last Updated:** 2026-08-29
**Status:** Active | **Review Cycle:** Monthly

---

## Vision

The Go counterpart to Rust's `interprocess`: one API for local connection-oriented byte streams
across Linux, macOS and Windows, with operating-system-enforced access control as the default
rather than an option. Small enough to audit in an afternoon.

---

## Phase 1: Core on Unix (Target: v0.1.0)

- [ ] `Name`, `NameKind`, `Filesystem`, `Namespaced`, `UserScoped` with validation
- [ ] `Listen` / `Dial` returning `net.Listener` / `net.Conn`
- [ ] Runtime-directory precedence with ownership and mode validation
- [ ] Stale-socket reclaim, restricted to owned sockets
- [ ] `PeerIdentity` via `SO_PEERCRED` / `getpeereid`
- [ ] `examples/echo`
- [ ] Suite verified locally on Linux and macOS with `-race`

14 acceptance criteria — see `docs/TESTING.md`.

---

## Phase 2: Windows (Target: v0.2.0)

- [ ] Named-pipe listener and dialer on `go-winio`
- [ ] Name resolution to `\\.\pipe\interprocess-go\<user-sid>\<id>`
- [ ] DACL restricted to the current user SID; logon session opt-in
- [ ] `PeerIdentity` via `GetNamedPipeClientProcessId`
- [ ] Suite verified locally on Windows

---

## Phase 3: Parity and Interop (Target: v0.3.0)

- [ ] Rust harness: Go listener ↔ Rust client and the reverse, all three platforms
- [ ] Conformance vector V4 asserted from both languages
- [ ] Parity table re-checked against the then-current `interprocess` release
- [ ] README security model, platform differences, compatibility matrix

---

## Phase 4: Framing (Target: v0.4.0)

- [ ] `WriteFrame` / `ReadFrame` per vector V1
- [ ] Reader robustness per vector V2
- [ ] Over-limit rejection without allocation
- [ ] Fuzz target with committed corpus
- [ ] `examples/request_reply` with a versioned JSON envelope

---

## v1.0.0 — API Freeze

Endpoint name resolution becomes a frozen contract. Before then it may change only for a security
fix, marked as such in `HISTORY.md`.

---

## Explicit Non-Goals

No TCP fallback · no datagram or message mode · no broker or discovery service · no process
management · no cryptographic authentication in the transport · no Linux abstract namespace.
