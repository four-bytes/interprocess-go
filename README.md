# interprocess-go

Local, connection-oriented byte streams over one API — Unix domain sockets on Linux and macOS,
named pipes on Windows. A deliberate Go counterpart to the Rust
[`interprocess`](https://crates.io/crates/interprocess) crate, so a Rust host and a Go agent can
talk to each other without either side inventing a private convention.

```go
name := localsocket.UserScoped("my-agent")

ln, err := localsocket.Listen(name, localsocket.ListenOptions{})
// ...
conn, err := localsocket.Dial(ctx, name, localsocket.DialOptions{})
```

`Listen` returns a `net.Listener`, `Dial` returns a `net.Conn`. Everything that already works with
those — `bufio`, `io`, your own protocol — works here.

## Why

A local TCP port on `127.0.0.1` is reachable by every process on the machine, and a token in a
file can be read by anything that can read files. Unix domain sockets and named pipes are
protected by the operating system itself: filesystem permissions on Unix, a DACL on Windows.

This library makes that protection the default rather than something you remember to configure,
and gives both platforms the same shape so application code stops branching on `GOOS`.

## Security model

| | Unix | Windows |
|---|---|---|
| Default access | current user | current user SID |
| Optional | — | current logon session |
| Runtime directory | owned by the current UID, mode `0700`, validated before use | not applicable |
| Socket / pipe | mode `0600` | restrictive DACL, never the default one |
| Network reachable | no | no remote pipe clients |
| TCP fallback | **none, ever** | **none, ever** |

Every runtime-directory candidate is validated for ownership and mode before it is used — an
explicitly passed one included. If none passes, `Listen` fails rather than falling back to
something weaker.

The Linux abstract namespace is **not** supported. Abstract sockets carry no filesystem
permissions, so any process in the same network namespace can connect to one; supporting it would
remove the only access control this library has on Unix. See [`GUIDELINES.md`](GUIDELINES.md).

## Platform support

| Platform | Transport | Status |
|---|---|---|
| Linux | Unix domain socket | Phase 1 |
| macOS | Unix domain socket | Phase 1 |
| Windows | Named pipe (`go-winio`) | Phase 2 |

## Framing

Local sockets are byte streams and do not preserve message boundaries. The `framing` subpackage
provides a 4-byte big-endian length prefix, with the maximum size checked **before** any
allocation:

```go
err := framing.WriteFrame(conn, payload, maxSize)
payload, err := framing.ReadFrame(conn, maxSize)
```

## Interoperating with Rust

Address the endpoint by its **literal path** on both sides. Rust's `GenericNamespaced` and this
library's `Namespaced` are library-private conventions and do not resolve to the same endpoint on
any platform — use `GenericFilePath` from Rust against the path a Go listener reports via
`LocalSocketName()`.

Better still, have the listening side publish its endpoint and the connecting side read it, rather
than both computing a name. See [`docs/INTEROP.md`](docs/INTEROP.md).

## Documentation

| File | Purpose |
|---|---|
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Package layout, name resolution, platform differences |
| [`docs/INTEROP.md`](docs/INTEROP.md) | Rust parity, the naming hazard, conformance vectors |
| [`docs/TESTING.md`](docs/TESTING.md) | Test architecture, vectors, fuzzing, CI matrix |
| [`GUIDELINES.md`](GUIDELINES.md) | Coding standards and security invariants |
| [`CONTRIBUTE.md`](CONTRIBUTE.md) | Issue → Branch → PR workflow |
| [`ROADMAP.md`](ROADMAP.md) | Phases and target versions |
| [`AGENTS.md`](AGENTS.md) | Conventions and security invariants for AI coding agents |
| [`HISTORY.md`](HISTORY.md) | Changelog |

## Status

Pre-`v0.1.0`. The public API is not frozen until `v1.0.0`; endpoint name resolution is pinned by
conformance vectors from `v1.0.0` on, and before then may change only for a security fix, marked
as such in the changelog.

## License

Apache-2.0 — see [`LICENSE`](LICENSE).
