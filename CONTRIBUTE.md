# Contributing to interprocess-go

## Golden Workflow

1. **Issue first** — every change starts with a GitHub issue stating purpose, scope and acceptance criteria.
2. **Branch** — `feat/GH-{issue}-description`, `fix/GH-{issue}-description`, etc.
3. **Code** — follow `GUIDELINES.md`.
4. **Test** — the local gate must pass (below). There is no CI safety net.
5. **PR** — open a pull request, link the issue, complete the checklist.
6. **Review** — merge only when green.
7. **Cleanup** — squash-merge, delete the branch.

## Commit Convention

[Conventional Commits](https://www.conventionalcommits.org/):

```
feat: description (#issue)
fix: description (#issue)
chore: description (#issue)
docs: description (#issue)
refactor: description (#issue)
test: description (#issue)
```

## Branch Naming

```
feat/GH-{nr}-short-description
fix/GH-{nr}-short-description
chore/GH-{nr}-short-description
docs/GH-{nr}-short-description
```

## The Local Gate

Quality is local. `.github/workflows/` carries deploy and release workflows only — nothing in
this repository verifies your change for you after you push.

```sh
gofmt -l .                          # must print nothing
go vet ./...                        # must be clean
go test -race ./...                 # must pass

GOOS=darwin  GOARCH=arm64 go build ./...   # cross-compile every supported platform
GOOS=darwin  GOARCH=amd64 go build ./...
GOOS=windows GOARCH=amd64 go build ./...   # from Phase 2 on
```

**Cross-compile every supported platform, always.** It needs no hardware, takes a second, and
catches the entire class of "this function does not exist on that OS" — which is exactly how a
`syscall.Getpeereid` that never existed reached `main` once. Compiling is not testing, but a
platform file that does not compile is not a testing gap, it is a broken build.

A change that touches platform-specific code must be run on that platform before merge. Claiming
"CI will catch it" is not available here, by design.

## Before You Open a PR

- The local gate above is green
- `HISTORY.md` has an entry under `[Unreleased]`
- No new dependency without a note in the PR describing why the standard library is insufficient

## Security-Relevant Changes

`CLAUDE.md` lists seven security invariants. A PR that touches permission handling, name
resolution, stale cleanup, or frame-length checking must:

- state which invariant it affects,
- add or extend a test that would fail if the invariant were violated,
- never relax a default to make a test pass.

If you believe an invariant is wrong, open an issue arguing that — do not weaken it inside a PR
about something else.

## Reporting a Vulnerability

Do not open a public issue. Use GitHub's private vulnerability reporting on this repository.
