# Interoperating with Rust

> Canonical design: `four-file-cloud/docs/interprocess-go-concept.md`.

## Compatibility with Rust

The transport abstractions are language-specific, but the connection is a byte stream. Rust `interprocess::local_socket` and `interprocess-go/local_socket` therefore need not share an API, but they must be able to use the same endpoint strategy and transmit the same data.

Interoperability contract for the File Core project:

```text
Transport
  Unix: Unix domain socket
  Windows: named pipe

Endpoint policy
  user-/logon-session-scoped

Framing
  4-byte unsigned big-endian length prefix

Payload
  UTF-8 JSON

Protocol
  versioned command/event schema
```

A CI test must verify both directions:

- Rust listener, Go client
- Go listener, Rust client
- Linux/macOS with UDS
- Windows with named pipes

---

## Parity with the Rust Crate

Surveyed against `interprocess` **2.4.3** (docs.rs, August 2026). Parity is the stated goal, so
the gaps have to be named rather than assumed away — and so does the one place where matching
Rust would be a mistake.

### Tier 1 — Must match. This is a contract, not a preference.

These are the surfaces where a Rust host and a Go agent talk to each other. A divergence here is
a bug that shows up as a connection that silently never works.

| Concern | Rust | Go | Status |
|---|---|---|---|
| Transport per platform | UDS / named pipe | UDS / named pipe | ✅ |
| Endpoint naming | `GenericFilePath`, `GenericNamespaced` | `Filesystem`, `Namespaced`, `UserScoped` | ✅ + `UserScoped` is an addition |
| Framing | not provided — application's job | `framing` subpackage | ✅ pinned by vector V1, asserted from both sides by V4 |
| Peer credentials | `local_socket::PeerCreds` | `PeerIdentity` (Decision 4) | ✅ — the survey confirms this belongs in the core, not an extension |

### Tier 2 — Should exist, expressed idiomatically

| Rust | Go equivalent | Status |
|---|---|---|
| `ListenerOptions` / `ConnectOptions` | `ListenOptions` / `DialOptions` | ✅ |
| `Name`, `NameType`, `ToFsName`, `ToNsName` | `Name`, `NameKind`, constructors | ✅ — Go needs no conversion traits |
| `os::windows::security_descriptor` | `PipeSecurity` + `AccessPolicy` | ⚠️ thinner; DACL construction still to be specified in code |
| `error` module | `errors.Is`/`As` sentinels | ✅ |
| `Incoming` iterator | `Accept()` in a `for` loop | ✅ idiomatic difference |

### Tier 3 — Deliberately not ported

Rust needs these because of the borrow checker and its async model. Go's runtime makes them moot,
and porting them would add API surface that buys nothing.

| Rust | Why not |
|---|---|
| `RecvHalf` / `SendHalf`, `split`/`reunite`, `ReuniteError` | A `net.Conn` is already safe for one reader and one writer concurrently |
| `ListenerNonblockingMode` | Blocking `Accept` in a goroutine is the Go model |
| `tokio` submodules | No async runtime to integrate with |
| `TryClone` | Go does not need fallible handle cloning at this layer |
| `ToWtf16`, `ImpersonationGuard` | Windows internals with no Go-side caller |

### Tier 4 — Real gaps, each with a recommendation

| Rust | Assessment |
|---|---|
| `unnamed_pipe` | **Skip.** `os.Pipe()` plus `exec.Cmd.ExtraFiles` is stdlib and idiomatic; a wrapper would add a name, not a capability. |
| `os::unix::fifo_file` | **Skip for v1.** `syscall.Mkfifo` is two lines. Add only if a caller asks. |
| `os::windows::named_pipe::PipeMode` (message mode) | **Deliberate divergence — do not port.** Message mode would preserve message boundaries on Windows and nowhere else, so a cross-platform protocol still needs length prefixes on Unix. Adopting it would mean two framing paths and one of them exercised on one platform only. Byte mode plus `framing` is uniform, and uniformity is the point. |
| `os::windows::ShareHandle` | **Later.** Relevant only if a supervisor hands a listening handle to a child process. Not in the file-core startup model. |

### The Parity Conflict Worth Naming

Rust's `GenericNamespaced` resolves to the **Linux abstract namespace** where available. Decision 3
declines it, on the grounds that abstract sockets carry no filesystem permissions.

**Full API parity and Decision 3 are mutually exclusive, and Decision 3 wins.** Copying a surface
because the reference implementation has it would import its weakest security property into a
library whose stated purpose is secure defaults. The parity that matters is Tier 1 — the bytes and
the endpoints two processes agree on — not a one-to-one map of every constructor.

`Namespaced(id)` in Go therefore resolves to a **filesystem** socket under a validated `0700`
runtime directory. A Rust peer must construct its name with `GenericFilePath` against that exact
path, never with `GenericNamespaced`.

**Connectivity is unaffected.** Rust reaches any filesystem socket through `ToFsName` /
`GenericFilePath`, so a Go listener and a Rust client meet without either side needing the
abstract namespace. Declining it removes an addressing *option*, not a capability.

**The actual hazard is naming, and it is not specific to Linux.** `Namespaced` in either library
is a library-private convention: Rust's `GenericNamespaced` resolves to Rust's directory on macOS,
Rust's pipe name on Windows, and an abstract socket on Linux. Ours resolves to ours. Two
implementations of "namespaced" never agree by accident on any platform — which is why
cross-language endpoints must be **discovered, not derived** (`file-core-client-concept.md`,
Endpoint Discovery), and why vector V4 pins a literal path on both sides rather than an
identifier.

---

