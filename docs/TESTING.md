# Testing

> Canonical design: `four-file-cloud/docs/interprocess-go-concept.md`.
> Vectors live in `testdata/` and are asserted from Go and from the Rust interop harness.

## Conformance Vectors

These are the artefacts that freeze the contract. They live in `testdata/` and are asserted by
tests on every platform; the Rust interop suite asserts the same bytes from the other side.

### V1 — Frame Encoding

`maxSize = 16777216` (16 MiB) for every vector. Payload bytes are shown in hex.

| Case | Payload | Encoded frame | Expectation |
|---|---|---|---|
| empty | *(none)* | `00 00 00 00` | round-trips to a zero-length, non-nil slice |
| single byte | `41` | `00 00 00 01 41` | round-trips |
| multi-byte UTF-8 | `c3 9f` (`ß`) | `00 00 00 02 c3 9f` | the library never inspects payload encoding |
| JSON envelope | `7b 7d` (`{}`) | `00 00 00 02 7b 7d` | framing is payload-agnostic |
| 1 MiB pattern | `i mod 251` for `i` in `[0,1048576)` | `00 10 00 00` + payload | exercises the large-write path |
| at the limit | 16777216 bytes | `01 00 00 00` + payload | accepted |
| over the limit | declared `01 00 00 01` | — | `ErrFrameTooLarge` **before allocation** |

Big-endian is normative. The over-the-limit case must be rejected from the prefix alone; a reader
that allocates first is a remote memory-exhaustion primitive, and this vector is what proves it
does not.

### V2 — Reader Robustness

| Case | Input | Expectation |
|---|---|---|
| fragmented prefix | 4-byte prefix delivered one byte per `Read` | assembles correctly |
| fragmented payload | payload delivered in 1-byte chunks | assembles correctly |
| clean EOF | stream ends exactly at a frame boundary | `io.EOF` |
| truncated payload | prefix declares 8, only 3 bytes follow, then EOF | `ErrShortFrame` (wrapping `io.ErrUnexpectedEOF`) |
| two frames, one `Read` | both frames in a single buffer | two successive `ReadFrame` calls succeed |

`io.EOF` at a frame boundary and a truncated frame must be distinguishable. Conflating them makes
a killed peer look like a clean shutdown.

### V3 — Name Validation and Resolution

| Input | Result |
|---|---|
| `file-core-agent` | valid |
| `file-core-agent-01` , `a.b_c-d` | valid |
| `` (empty) | `ErrInvalidName` |
| `has/slash` , `has\backslash` | `ErrInvalidName` |
| `has space` , `ümlaut` | `ErrInvalidName` — ASCII `[A-Za-z0-9._-]` only |
| 200-char identifier | resolves via the Decision 6 rule; the vector pins the exact output string |

Resolution shapes, asserted per platform:

```text
Linux    <runtime_dir>/interprocess-go/<id>.sock      dir 0700, socket 0600
macOS    <runtime_dir>/interprocess-go/<id>.sock      dir 0700, socket 0600
Windows  \\.\pipe\interprocess-go\<user-sid>\<id>     DACL: current user SID only
```

### V4 — Interop

Go listener ↔ Rust client and Rust listener ↔ Go client, on every platform in the support matrix,
exchanging every V1 vector and asserting byte equality in both directions. A vector that only one
language can produce is not a contract.

---

## Test Strategy

### Unit Tests

- name validation and deterministic resolution
- length, character set, and collision handling
- options and secure defaults
- framing with empty, small, large, fragmented, and invalid frames

### Platform Tests

- echo between listener and client
- multiple parallel clients
- listener close terminates `Accept`
- context cancel during dial
- restart after a clean close
- restart after a simulated crash, i.e. a stale Unix socket
- permission/DACL tests, run on a real machine of that platform

### Interop Tests

- Rust `interprocess` ↔ Go `interprocess-go`
- byte-exact test vector for the reference frames
- versioned test scenarios in the repository, not just manual demos

### Platform Matrix — Run Locally

There is no test CI. `.github/workflows/` carries deploy and release workflows only; runner
minutes are not spent re-running a gate that already passed on the author's machine.

```text
Linux    required before merging any change to Unix code
macOS    required before merging any change to Unix code
Windows  required before merging any change to named-pipe code
Go       the supported Go versions
Rust     stable toolchain, for the interop harness
```

A change confined to documentation or to one platform's files needs only that platform. A change
to shared code — name validation, framing, options — needs all of them. "Someone else will run it"
is how a platform bug ships.

---

## Implementation Phases

Each phase is one issue, one branch, one PR, and is independently verifiable. A phase is done when
**every** criterion below passes — a criterion that cannot be asserted by a test is not a
criterion.

### Phase 1: Core on Unix

**Scope:** Linux and macOS. No Windows code, no framing.

| # | Acceptance criterion |
|---|---|
| 1.1 | `Listen` and `Dial` work for `Filesystem`, `Namespaced` and `UserScoped` on Linux and macOS. |
| 1.2 | Name validation matches vector **V3** exactly, including the 200-character case and the Decision 6 truncation output. |
| 1.3 | Runtime directory resolution follows Decision 2 in order, and **every** candidate is rejected unless it exists, is a directory, is owned by the current UID, and has no group or world write bit. A test sets `$TMPDIR` to a `0777` directory on Linux and asserts it is skipped. |
| 1.4 | With no valid candidate, `Listen` fails with `ErrNoRuntimeDir` and creates nothing. The candidate chain is injected in the test — driving it through the environment leaves `/run/user/$UID` in the list on every systemd host, so the test would skip itself exactly where it matters. |
| 1.5 | After `Listen`, the runtime directory is mode `0700` and the socket file is mode `0600`, asserted by `os.Stat`. |
| 1.6 | Stale-socket reclaim follows the documented six steps. A test writes a **regular file** at the socket path and asserts `ErrStaleCleanupUnsafe` — only an owned socket is ever removed. |
| 1.7 | `Close()` removes the listener's own socket file **with no options set**; a second `Listen` on the same name then succeeds. `KeepOnClose` opts out. A default that cannot restart is a defect, not caution. |
| 1.8 | `Close()` causes a blocked `Accept()` to return, and the returned error satisfies `errors.Is(err, net.ErrClosed)`. |
| 1.9 | `Dial` honours context cancellation and `DialOptions.Timeout`; a cancelled dial returns promptly and leaks no goroutine (`goleak` or an equivalent check). |
| 1.10 | `PeerIdentity()` returns the correct UID and GID (`SO_PEERCRED` on Linux, `getpeereid` on macOS). |
| 1.11 | Echo across 64 concurrent clients, 1 MiB per client, with no data corruption and no race under `-race`. |
| 1.12 | Restart after a simulated crash (process killed, socket file left behind) succeeds. |
| 1.13 | `examples/echo` builds and runs on both platforms. |
| 1.14 | Suite verified locally on Linux with `-race`. macOS: cross-compiled and vetted for `darwin/arm64` and `darwin/amd64`; **runtime behaviour unverified** — no macOS hardware available. See the platform matrix note below. |

### Phase 2: Windows

| # | Acceptance criterion |
|---|---|
| 2.1 | `Listen` and `Dial` work over named pipes via `go-winio`, returning `net.Listener` and `net.Conn`. |
| 2.2 | Names resolve to `\\.\pipe\interprocess-go\<user-sid>\<id>` exactly as vector **V3** states. |
| 2.3 | The pipe DACL grants the current user SID only. A test running as the same user connects; the DACL is read back and asserted to contain no `Everyone`, `Anonymous`, or `Authenticated Users` entry. |
| 2.4 | `AccessCurrentLogonSession` additionally restricts to the logon SID, and is **not** the default. |
| 2.5 | `Filesystem(path)` returns `ErrUnsupportedName` on Windows — explicitly, never a silent fallback. |
| 2.6 | `PeerIdentity()` returns the client SID (`GetNamedPipeClientProcessId`, then the process token). |
| 2.7 | Every Phase 1 behavioural test that is not Unix-specific passes unchanged on Windows: echo, 64 concurrent clients, `Close` unblocks `Accept`, dial cancellation, restart. |
| 2.8 | Suite verified locally on Windows. |

### Phase 3: Parity and Interop

| # | Acceptance criterion |
|---|---|
| 3.1 | A Rust harness using `interprocess` 2.4.x connects to a Go listener, and a Go client connects to a Rust listener, on all three platforms. |
| 3.2 | Both directions exchange every **V1** vector and assert byte equality — vector **V4**. |
| 3.3 | The Rust side addresses the endpoint with `GenericFilePath` against the literal path on Unix; the harness contains a comment stating why `GenericNamespaced` would fail. |
| 3.4 | The Tier table in this document is re-checked against the then-current `interprocess` release and updated in the same PR. |
| 3.5 | README carries the security model, the platform differences, a minimal example, and the compatibility matrix. |

### Phase 4: Framing

| # | Acceptance criterion |
|---|---|
| 4.1 | `WriteFrame` and `ReadFrame` produce and accept every **V1** vector byte-exactly. |
| 4.2 | Every **V2** reader case passes: fragmented prefix, fragmented payload, clean `io.EOF` at a boundary, `ErrShortFrame` on truncation, two frames in one buffer. |
| 4.3 | A declared length above `maxSize` returns `ErrFrameTooLarge` **without allocating**, proven by a test whose declared length would exhaust memory if allocated first. |
| 4.4 | `io.EOF` and `ErrShortFrame` are distinguishable by `errors.Is`. |
| 4.5 | A fuzz target over `ReadFrame` runs locally with no crashers, and its corpus is committed. |
| 4.6 | `examples/request_reply` demonstrates a versioned JSON envelope over the framing. |

---

