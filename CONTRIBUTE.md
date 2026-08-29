# Contributing to interprocess-go

## Golden Workflow

1. **Issue first** — every change starts with a GitHub issue stating purpose, scope and acceptance criteria.
2. **Branch** — `feat/GH-{issue}-description`, `fix/GH-{issue}-description`, etc.
3. **Code** — follow `GUIDELINES.md`.
4. **Test** — `go test -race ./...` must pass on your platform, and CI on all three.
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

## Before You Open a PR

- `gofmt -l .` prints nothing
- `go vet ./...` is clean
- `go test -race ./...` passes
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
