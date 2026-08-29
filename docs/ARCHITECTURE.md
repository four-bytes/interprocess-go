# Architecture

> Canonical design: `four-file-cloud/docs/interprocess-go-concept.md`.
> This file mirrors the parts an implementer needs in-repo.

## Package Structure

```text
interprocess-go/
  local_socket/
    name.go
    name_unix.go
    name_windows.go
    listener.go
    stream.go
    options.go
    security.go
    peer.go
    local_socket_unix.go
    local_socket_windows.go
    cleanup_unix.go
    errors.go

  framing/                  # subpackage of the same module (Decision 5)
    length_prefix.go
    reader.go
    writer.go
    limits.go

  internal/
    platform/
    testutil/

  examples/
    echo/
    request_reply/
    rust_interop/

  .github/workflows/
    test.yml
    release.yml
```

The platform files use build tags:

```go
//go:build unix
```

and:

```go
//go:build windows
```

---

## Public API

### Package `local_socket`

```go
package localsocket

import (
    "context"
    "net"
)

type NameKind uint8

const (
    NameFilesystem NameKind = iota
    NameNamespaced
)

type Name struct {
    Kind  NameKind
    Value string
}

func Filesystem(path string) Name
func Namespaced(identifier string) Name
func UserScoped(identifier string) Name

func Listen(name Name, options ListenOptions) (net.Listener, error)
func Dial(ctx context.Context, name Name, options DialOptions) (net.Conn, error)
```

The return types are deliberately `net.Listener` and `net.Conn`. That lowers the learning curve and allows direct use with existing Go code, e.g. `bufio`, `io`, `net/http`-style adapters, or custom protocols.

Two concrete interfaces expose additional metadata. Both are part of the **v1 core**
(Decision 4); every value returned by `Listen` and `Dial` satisfies them:

```go
type Listener interface {
    net.Listener
    LocalSocketName() Name
}

type Conn interface {
    net.Conn
    PeerIdentity() (PeerIdentity, error)
}

type PeerIdentity struct {
    PID int    // 0 if the platform does not report it
    UID uint32 // Unix only
    GID uint32 // Unix only
    SID string // Windows only
}
```

`PeerIdentity` returns `ErrPeerIdentityUnsupported` where the platform cannot supply it. Callers
must handle that error rather than reading the zero value as "nobody".

Application code may still restrict itself to `net.Listener` and `net.Conn`; the extra methods are
reachable by type assertion and cost nothing to ignore.

---

## Names and Resolution

### Core Idea

A local socket name is not identical to an OS path. Applications should express a portable intent that is resolved only in the respective platform implementation.

```go
agentName := localsocket.UserScoped("file-core-agent")
```

### Name Kinds

| Kind | Meaning | Primary use |
|---|---|---|
| `Filesystem(path)` | Explicit Unix socket path | Unix-specific or controlled deployments |
| `Namespaced(id)` | Platform-neutral local name | Services with controlled security |
| `UserScoped(id)` | Name and access for the current user or logon context | Default for desktop agents |

### Platform Resolution

| Name | Linux/macOS | Windows |
|---|---|---|
| `Filesystem(path)` | UDS at `path` | Unsupported, or a clear error |
| `Namespaced(id)` | `<runtime_dir>/interprocess-go/<id>.sock` | `\\.\pipe\interprocess-go\<id>` |
| `UserScoped(id)` | same path; runtime dir per Decision 2 | `\\.\pipe\interprocess-go\<user-sid>\<id>` |

`UserScoped` is the recommended default. These shapes are pinned by conformance vector V3 — they
are a frozen contract from v1.0.0 on, not an implementation detail. Before v1.0.0 they may still
change for a security fix, with a changelog entry marked as such.

### Validation

- Namespace IDs are restricted, e.g. to ASCII letters, digits, `-`, `_`, and `.`.
- No path separators in `Namespaced` or `UserScoped`.
- Bounded maximum length before OS resolution.
- Every normalization is documented and deterministic.
- **Long-name rule (Decision 6).** Platform endpoint names are bounded: `sun_path` is 108 bytes on
  Linux and 104 on macOS, and a pipe name is 256 characters. When the resolved endpoint would
  exceed the platform limit, the identifier is replaced by:

  ```text
  <first 16 bytes of the identifier><"-"><first 8 lowercase hex of sha256(full identifier)>
  ```

  Deterministic, stable across processes and platforms, and covered by a conformance vector. The
  truncation keeps the name recognisable to an operator; the hash suffix carries the uniqueness.

---

## Options and Security

```go
type ListenOptions struct {
    Access              AccessPolicy
    RuntimeDir          string
    ReclaimStale        bool
    KeepOnClose         bool
    PipeSecurity        *PipeSecurity
    MaxInstances        int
}

type DialOptions struct {
    Timeout time.Duration
}

type AccessPolicy uint8

const (
    AccessCurrentUser AccessPolicy = iota // default (Decision 1)
    AccessCurrentLogonSession
    AccessCustom
)
```

`AccessCurrentUser` is the zero value and therefore what a caller passing an empty `ListenOptions`
gets. A secure default must not require an explicit opt-in.

### Secure Defaults

| Topic | Unix | Windows |
|---|---|---|
| Default access | current user | current user SID; logon session opt-in (Decision 1) |
| Directory | owned by current user, mode `0700` | not applicable |
| Socket/pipe | socket file mode `0600` | restrictive DACL |
| Network access | none | no remote pipe clients |
| Cleanup | only verified, own stale socket files | the OS manages pipe lifetime |

The library must not use a silent TCP fallback. A local endpoint has to stay local.

### Unix Cleanup

Before binding to a file socket:

1. Create or verify the parent runtime directory.
2. Validate ownership and mode of the directory.
3. If an entry exists at the socket path: validate file type and owner.
4. Remove only a socket file that is classified as stale and owned by us.
5. Bind the UDS.
6. Set and verify restrictive mode bits.

On `Close()` the listener removes its own socket file by default (`KeepOnClose` opts out). Rust `interprocess` also treats automatic name release on listener drop as part of local socket semantics; the Go variant should satisfy that expectation with an explicit `Close()`.

### Windows Security

The Windows implementation should not reimplement the entire named-pipe layer itself. It builds on a maintained named-pipe base such as `Microsoft/go-winio` and focuses on:

- correct name construction under `\\.\pipe\...`
- an explicit security descriptor / DACL strategy
- access only for the intended user SID, optionally plus the logon SID
- no broadly open default DACL
- clear mapping of Windows errors into package errors

---

## Platform Implementations

### Linux and macOS

- `net.Listen("unix", path)` and `net.Dialer.DialContext(ctx, "unix", path)`
- Runtime directory as the secure base path, always with a private `0700` subdirectory
- File-based UDS only (Decision 3)

**Runtime directory precedence (Decision 2)** — first hit wins, identical logic on Linux and
macOS, so there is one code path rather than two:

```text
1. ListenOptions.RuntimeDir        explicit
2. $XDG_RUNTIME_DIR                Linux standard; honoured on macOS if the user set it
3. $TMPDIR                         darwin ONLY — per-user /var/folders/../T, mode 0700
4. /run/user/$UID                  Linux, when the variable is unset but the directory exists
5. os.UserCacheDir()               last resort, always with a 0700 subdirectory
```

**Every candidate is validated before use, whatever its source:** it must exist, be a directory,
be owned by the current UID, and have no group or world write bit. A candidate that fails is
skipped, not repaired. If none survives, `Listen` fails with `ErrNoRuntimeDir` rather than falling
back to something weaker.

**`$TMPDIR` is gated to Darwin, and that is not cosmetic.** The two platforms are not
comparable here:

| | `$TMPDIR` | Safe as a socket parent? |
|---|---|---|
| macOS | `/var/folders/<xx>/<yyy>/T/`, per user, mode `0700` | yes |
| Linux | usually unset; conventionally means `/tmp`, world-writable, sticky | **no** |

An unconditional `$TMPDIR` step would therefore place a Linux socket in a world-writable
directory whenever the variable happened to be set — the exact failure the library exists to
prevent. The platform gate states the intent; the ownership-and-mode validation is what actually
enforces it, and it applies to steps 1 and 2 as well.

That macOS may purge `$TMPDIR` is harmless: a socket is ephemeral and is recreated on the next
`Listen`.

**Never** `/tmp` directly.

### Linux Abstract Namespace — Declined

**Decision 3: the abstract namespace is not supported, in v1 or later.**

Abstract sockets carry no filesystem permissions. Any process in the same network namespace may
connect to one, regardless of user. That is not a weaker variant of the file-based socket — it is
the removal of the only access control this library has on Unix, and it contradicts the stated
goal of secure defaults directly.

*Fallback:* file-based UDS under a `0700` runtime directory, which works in every environment
including containers. *Reopens if* a supported deployment provides no writable runtime directory
at all — at which point the answer is likely a mounted `tmpfs`, not an abstract socket.
*Owner:* maintainer.

### Windows

- Named pipes as byte streams, compatible with `net.Conn`
- `Listen` creates a named-pipe listener
- `Dial` uses a context- and timeout-aware pipe connection
- Security descriptor derived from `AccessPolicy`
- Errors for unsupported name kinds must be explicit and machine-readable

---

## Framing (`framing` subpackage)

Local sockets are byte streams and do not preserve message boundaries. The `framing` subpackage
provides the length prefix (Decision 5 — a subpackage of the same module, since it needs only `io`
and `encoding/binary`):

```go
func WriteFrame(w io.Writer, payload []byte, maxSize uint32) error
func ReadFrame(r io.Reader, maxSize uint32) ([]byte, error)
```

Proposal for the reference implementation:

```text
4 bytes: unsigned big-endian payload length
N bytes: payload
```

Requirements:

- check maximum payload size before allocation
- handle partial reads and partial writes correctly
- distinguish EOF cleanly
- make no assumption about JSON, Protobuf, or other payload formats
- tests with fragmented reads/writes and oversized frames

---

## Error Design

```go
var (
    ErrInvalidName        = errors.New("invalid local socket name")
    ErrUnsupportedName    = errors.New("unsupported name type on this platform")
    ErrPermissionDenied   = errors.New("local socket permission denied")
    ErrAlreadyInUse       = errors.New("local socket already in use")
    ErrStaleCleanupUnsafe = errors.New("refusing unsafe stale socket cleanup")
    ErrNoRuntimeDir       = errors.New("no runtime directory passed ownership and mode validation")

    ErrPeerIdentityUnsupported = errors.New("peer identity unavailable on this platform")

    // framing
    ErrFrameTooLarge = errors.New("frame exceeds configured maximum")
    ErrShortFrame    = errors.New("frame truncated before declared length")
)
```

Platform errors must remain inspectable via `errors.Is` and `errors.As`. The library must not swallow security-relevant original errors.

---

## Decisions

Closed 2026-08-29. Nothing in this document is open; a change from here is a versioned amendment,
not a gap.

| # | Question | Decision | Rationale |
|---|---|---|---|
| 1 | `UserScoped`: user SID only, or also logon SID? | **User SID by default**, logon session opt-in via `AccessCurrentLogonSession` | Logon-SID scoping breaks legitimate cases — a service, or a second desktop session of the same user, cannot connect, and RDP and fast user switching change the logon session. User-SID scoping is also the exact analogue of Unix `0600`, which keeps one meaning across platforms. |
| 2 | macOS runtime-directory precedence | `RuntimeDir` → `$XDG_RUNTIME_DIR` → `$TMPDIR` *(Darwin only)* → `/run/user/$UID` → `os.UserCacheDir()`; **every candidate validated for owner and mode** | macOS `$TMPDIR` is per-user and `0700`; Linux `$TMPDIR` conventionally means world-writable `/tmp` and must never be used. One precedence list with one platform gate, plus validation that does not trust any source — including an explicit `RuntimeDir`. |
| 3 | Linux abstract namespace | **Declined, permanently** | Abstract sockets have no filesystem permissions; supporting them would delete the only Unix access control this library has. Fallback, trigger and owner are recorded in the platform section. |
| 4 | `PeerIdentity()` in the v1 core? | **Yes**, with `ErrPeerIdentityUnsupported` | File permissions say who *may* connect; peer credentials say who *did*. That is the second half of the security story and what makes an application-level handshake verifiable. As an extension, most callers would skip it. |
| 5 | `framing`: subpackage or separate module? | **Subpackage** | It needs only `io` and `encoding/binary` — zero dependencies — so the stated reason for a split (keeping the core dependency minimal) does not apply. Module boundaries should follow dependency boundaries, not conceptual ones, and a second module doubles release overhead for ~200 lines. |
| 6 | Long-name handling | Truncate to 16 bytes + `-` + 8 hex of `sha256(identifier)` | Endpoint names are length-bounded per platform. The rule is deterministic, keeps the name recognisable to an operator, and is pinned by a conformance vector. |

---

