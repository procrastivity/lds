// Package asset implements the contract's asset-resolution chain (C5.1):
// user override -> shipped default -> embedded last-resort fallback. An
// override and a shipped default at the same relative path are the same
// logical asset — the override fully replaces the default, never merged
// (shadow by name, C5.2). Assets are inputs, never state (C5.3): this
// package only ever reads.
package asset

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	rootassets "github.com/procrastivity/lds/assets"
)

// appName names both the XDG config subdirectory and the installed share
// subtree: $XDG_CONFIG_HOME/lds and <prefix>/share/lds/assets.
const appName = "lds"

// executable is os.Executable, indirected so tests can point default-dir
// resolution at a fake <prefix>/bin/lds without needing a real
// installed layout on disk (C2.6).
var executable = os.Executable

// Source names which link of the resolution chain satisfied a Resolve call.
type Source int

const (
	// SourceOverride is the user-override directory, $XDG_CONFIG_HOME/lds.
	SourceOverride Source = iota
	// SourceDefault is the shipped default directory, <prefix>/share/lds/assets.
	SourceDefault
	// SourceEmbedded is the last-resort fallback compiled into the binary.
	SourceEmbedded
	// SourceProject is an explicitly selected project's .lds/assets directory.
	SourceProject
)

func (s Source) String() string {
	switch s {
	case SourceOverride:
		return "override"
	case SourceDefault:
		return "default"
	case SourceEmbedded:
		return "embedded"
	case SourceProject:
		return "project"
	default:
		return "unknown"
	}
}

// Resolved is one asset located by Resolve or ResolveIn.
type Resolved struct {
	Source Source
	// Path is the filesystem path the asset was read from; empty when
	// Source is SourceEmbedded (the asset lives in the compiled binary,
	// not on disk).
	Path string

	data []byte
}

// Bytes returns the asset's content.
func (r Resolved) Bytes() []byte {
	return r.data
}

// OverrideDir returns $XDG_CONFIG_HOME/lds, the user-override directory.
func OverrideDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("asset: resolving XDG_CONFIG_HOME fallback: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, appName), nil
}

// DefaultDir returns <prefix>/share/lds/assets, resolved relative to
// the running executable: the binary is assumed to sit at
// <prefix>/bin/lds, with a sibling <prefix>/share/lds/assets —
// the layout buildGoModule's $out already guarantees, so no build-time
// patching is needed (C5.1).
func DefaultDir() (string, error) {
	exe, err := executable()
	if err != nil {
		return "", fmt.Errorf("asset: locating running executable: %w", err)
	}
	prefix := filepath.Dir(filepath.Dir(exe)) // .../bin/lds -> .../bin -> <prefix>
	return filepath.Join(prefix, "share", appName, "assets"), nil
}

// Resolve locates relativeName by the override -> default -> embedded
// chain and returns its content. Shadow by name: an override present at
// relativeName fully replaces a shipped default at the same relative path.
func Resolve(relativeName string) (Resolved, error) {
	if r, ok, err := ReadOverride(relativeName); err != nil {
		return Resolved{}, err
	} else if ok {
		return r, nil
	}
	return ReadDefault(relativeName)
}

// ResolveIn resolves a known shipped asset through project -> override ->
// default -> embedded. An empty projectRoot disables the project link. A
// nonempty root must name an existing directory; relative roots are relative
// to the invocation cwd, with no discovery or environment-based selection.
// The selected root is canonicalized, and project provenance uses that path.
//
// relativeName must be a portable, slash-relative file path, with no empty,
// dot, dot-dot, backslash, colon or NUL components. Eligibility comes from a
// regular file in the embedded or installed default tree, excluding top-level
// Go package infrastructure. Project and user files cannot introduce names.
// This does not change Resolve's name or strict XDG error behavior.
//
// Only a missing project path falls through. Symlinks anywhere below the
// canonical root, nonregular files, invalid roots and other filesystem errors
// refuse resolution. Unsafe names, symlinks and nonregular paths wrap
// fs.ErrInvalid; unknown names wrap fs.ErrNotExist; filesystem errors retain
// their underlying identity.
// Empty or malformed content is returned unchanged for the consumer to parse,
// never retried against a lower-precedence source.
//
// The caller must trust the selected directory and its writers. These checks
// do not protect against concurrent path replacement; this is not a sandbox.
func ResolveIn(projectRoot, relativeName string) (Resolved, error) {
	// Validate before any cleaning or joining can erase unsafe components.
	if !fs.ValidPath(relativeName) || relativeName == "." ||
		strings.ContainsAny(relativeName, "\\:\x00") || !filepath.IsLocal(relativeName) {
		return Resolved{}, fmt.Errorf("asset: invalid relative name %q: %w", relativeName, fs.ErrInvalid)
	}

	var root string
	if projectRoot != "" {
		absolute, err := filepath.Abs(projectRoot)
		if err != nil {
			return Resolved{}, fmt.Errorf("asset: project root %q: %w", projectRoot, err)
		}
		root, err = filepath.EvalSymlinks(absolute)
		if err != nil {
			return Resolved{}, fmt.Errorf("asset: project root %q: %w", projectRoot, err)
		}
		info, err := os.Stat(root)
		if err != nil {
			return Resolved{}, fmt.Errorf("asset: project root %q: %w", projectRoot, err)
		}
		if !info.IsDir() {
			return Resolved{}, fmt.Errorf("asset: project root %q is not a directory: %w", projectRoot, fs.ErrInvalid)
		}
	}

	if !knownShippedName(relativeName) {
		return Resolved{}, fmt.Errorf("asset: %q is not a known shipped asset: %w", relativeName, fs.ErrNotExist)
	}
	if root == "" {
		return Resolve(relativeName)
	}

	parts := strings.Split("."+appName+"/assets/"+relativeName, "/")
	path := root
	for i, part := range parts {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			return Resolve(relativeName)
		}
		if err != nil {
			return Resolved{}, fmt.Errorf("asset: project path %q: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return Resolved{}, fmt.Errorf("asset: project path %q is a symlink: %w", path, fs.ErrInvalid)
		}
		if i < len(parts)-1 {
			if !info.IsDir() {
				return Resolved{}, fmt.Errorf("asset: project path %q is not a directory: %w", path, fs.ErrInvalid)
			}
		} else if !info.Mode().IsRegular() {
			return Resolved{}, fmt.Errorf("asset: project path %q is not a regular file: %w", path, fs.ErrInvalid)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Resolved{}, fmt.Errorf("asset: reading project asset %q: %w", path, err)
	}
	return Resolved{Source: SourceProject, Path: path, data: data}, nil
}

// knownShippedName checks only shipped sources, never project or XDG content.
// Installed-only names are eligible without requiring a matching embedded copy.
func knownShippedName(name string) bool {
	if !strings.Contains(name, "/") && strings.HasSuffix(name, ".go") {
		return false
	}
	if info, err := fs.Stat(rootassets.FS, name); err == nil && info.Mode().IsRegular() {
		return true
	}
	if dir, err := DefaultDir(); err == nil {
		if info, err := os.Stat(filepath.Join(dir, name)); err == nil && info.Mode().IsRegular() {
			return true
		}
	}
	return false
}

// ReadOverride reads relativeName from the user-override directory only,
// reporting whether it was present. Config uses this directly, since its
// override and default filenames differ (config.yaml vs.
// config.default.yaml) and so can't share one Resolve call.
func ReadOverride(relativeName string) (Resolved, bool, error) {
	dir, err := OverrideDir()
	if err != nil {
		return Resolved{}, false, err
	}
	path := filepath.Join(dir, relativeName)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return Resolved{}, false, nil
	}
	if err != nil {
		return Resolved{}, false, fmt.Errorf("asset: inspect override %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return Resolved{}, false, fmt.Errorf("asset: override %q is not a regular file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Resolved{}, false, fmt.Errorf("asset: read override %q: %w", path, err)
	}
	return Resolved{Source: SourceOverride, Path: path, data: data}, true, nil
}

// ReadDefault reads relativeName from the shipped default directory, falling
// back to the embedded last-resort copy when no installed share tree has it.
func ReadDefault(relativeName string) (Resolved, error) {
	if dir, err := DefaultDir(); err == nil {
		path := filepath.Join(dir, relativeName)
		if data, ok := readIfExists(path); ok {
			return Resolved{Source: SourceDefault, Path: path, data: data}, nil
		}
	}

	if data, err := rootassets.FS.ReadFile(relativeName); err == nil {
		return Resolved{Source: SourceEmbedded, data: data}, nil
	}

	return Resolved{}, fmt.Errorf("asset: %q not found in default directory or embedded fallback", relativeName)
}

func readIfExists(path string) ([]byte, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}
