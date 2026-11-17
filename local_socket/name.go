// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Four Bytes

package localsocket

import (
	"crypto/sha256"
	"encoding/hex"
	"runtime"
)

// NameKind distinguishes how a Name is resolved to a platform endpoint.
type NameKind uint8

const (
	// NameFilesystem is an explicit filesystem path (Unix domain socket path).
	NameFilesystem NameKind = iota

	// NameNamespaced is a platform-neutral local identifier resolved under a
	// validated runtime directory.
	NameNamespaced
)

// Name is a portable local-socket endpoint. Applications express intent with a
// constructor and let the platform implementation resolve it.
type Name struct {
	Kind  NameKind
	Value string
}

// Filesystem returns a Name for an explicit Unix socket path. It is intended
// for Unix-specific or controlled deployments.
func Filesystem(path string) Name {
	return Name{Kind: NameFilesystem, Value: path}
}

// Namespaced returns a platform-neutral local name resolved under the
// validated runtime directory (see ARCHITECTURE.md, Decision 2).
func Namespaced(identifier string) Name {
	return Name{Kind: NameNamespaced, Value: identifier}
}

// UserScoped returns a name scoped to the current user. On Unix it resolves to
// the same path as Namespaced; on Windows it is additionally scoped to the
// user SID. It is the recommended default for desktop agents.
func UserScoped(identifier string) Name {
	return Name{Kind: NameNamespaced, Value: identifier}
}

// validateIdentifier enforces the V3 name rules: non-empty and ASCII
// [A-Za-z0-9._-] only. Length is not rejected here — an over-long identifier
// is truncated by the Decision 6 rule at resolution time.
func validateIdentifier(identifier string) error {
	if identifier == "" {
		return ErrInvalidName
	}
	for i := 0; i < len(identifier); i++ {
		c := identifier[i]
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '.' || c == '_' || c == '-') {
			return ErrInvalidName
		}
	}
	return nil
}

// truncateIdentifier implements the Decision 6 long-name rule: when a resolved
// endpoint would exceed the platform limit, the identifier is replaced by the
// first 16 bytes of the identifier, a hyphen, and the first 8 lowercase hex
// digits of sha256(full identifier). Deterministic across processes and
// platforms; pinned by conformance vector V3.
func truncateIdentifier(identifier string) string {
	prefix := identifier
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	sum := sha256.Sum256([]byte(identifier))
	return prefix + "-" + hex.EncodeToString(sum[:4])
}

// maybeTruncate applies the Decision 6 rule only when the fully resolved socket
// path would exceed the platform sun_path limit.
func maybeTruncate(identifier, dir string) string {
	if len(joinSocketPath(dir, identifier)) <= maxSocketPathLength() {
		return identifier
	}
	return truncateIdentifier(identifier)
}

// maxSocketPathLength returns the usable sun_path length: 108 bytes on Linux
// and 104 on macOS, each minus one for the terminating NUL.
func maxSocketPathLength() int {
	if runtime.GOOS == "darwin" {
		return 103
	}
	return 107
}

// joinSocketPath joins a directory and identifier into the socket path shape
// pinned by vector V3: <dir>/<id>.sock.
func joinSocketPath(dir, identifier string) string {
	return dir + "/" + identifier + ".sock"
}
