# interprocess-go — Change History

## [Unreleased]

### Added
- Repository skeleton: Apache-2.0 license, README, CLAUDE.md, CONTRIBUTE.md, GUIDELINES.md,
  ROADMAP.md, `docs/` set, `.github/` templates, CI matrix over Linux/macOS/Windows.
- Design carried over from `four-file-cloud/docs/interprocess-go-concept.md` at D5
  (Implementable): six closed API decisions, conformance vectors V1-V4, per-phase acceptance
  criteria.

### Technical Details
- Module path: `github.com/four-bytes/interprocess-go`
- Go 1.24 minimum; standard library only on Unix, `Microsoft/go-winio` on Windows
- No code yet — Phase 1 (Unix core) is the first implementation issue
