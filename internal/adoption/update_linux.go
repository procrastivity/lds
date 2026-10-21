//go:build linux

package adoption

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

const (
	updateMaxBytes = 8 << 20
	updateMetadata = "no-xattrs;parent-no-default-acl"
)

func updateParts(relative string) ([]string, error) {
	if relative == "" || !utf8.ValidString(relative) || path.Clean(relative) != relative || strings.HasPrefix(relative, "/") || strings.ContainsAny(relative, "\\:") {
		return nil, ErrUnsafeTarget
	}
	parts := strings.Split(relative, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return nil, ErrUnsafeTarget
		}
		for _, c := range part {
			if unicode.IsControl(c) {
				return nil, ErrUnsafeTarget
			}
		}
	}
	return parts, nil
}

// The caller owns root and every descriptor returned in opened. The final
// parent is root for a root-level leaf. No target parent is ever created.
func updateParent(root *os.File, parts []string) (*os.File, []ParentIdentity, []*os.File, error) {
	parent := root
	var identities []ParentIdentity
	var opened []*os.File
	for i, component := range parts[:len(parts)-1] {
		next, err := openChild(parent, component)
		if err != nil {
			return nil, nil, opened, err
		}
		opened = append(opened, next)
		info, err := next.Stat()
		if err != nil {
			return nil, nil, opened, err
		}
		if !info.IsDir() {
			return nil, nil, opened, ErrUnsafeTarget
		}
		identity, err := rootIdentity(next)
		if err != nil {
			return nil, nil, opened, err
		}
		identities = append(identities, ParentIdentity{Path: strings.Join(parts[:i+1], "/") + "/", Identity: identity})
		parent = next
	}
	return parent, identities, opened, nil
}

func updateClose(files []*os.File) {
	for _, f := range files {
		_ = f.Close()
	}
}

func updateNoDefaultACL(parent *os.File) error {
	_, err := unix.Fgetxattr(int(parent.Fd()), "system.posix_acl_default", nil)
	if errors.Is(err, unix.ENODATA) {
		return nil
	}
	if err == nil {
		return fmt.Errorf("%w: parent has a default ACL", ErrUnsafeTarget)
	}
	return fmt.Errorf("%w: cannot check parent default ACL: %v", ErrUnsafeTarget, err)
}

func updateNoXattrs(file *os.File) error {
	n, err := unix.Flistxattr(int(file.Fd()), nil)
	if err != nil {
		return fmt.Errorf("%w: cannot check file xattrs: %v", ErrUnsafeTarget, err)
	}
	if n != 0 {
		return fmt.Errorf("%w: file has xattrs or an extended ACL", ErrUnsafeTarget)
	}
	return nil
}

func updateState(file, parent *os.File) (UpdateFileState, error) {
	var state UpdateFileState
	var st unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &st); err != nil {
		return state, err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Size > updateMaxBytes {
		return state, ErrUnsafeTarget
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return state, err
	}
	data, err := io.ReadAll(io.LimitReader(file, updateMaxBytes+1))
	if err != nil {
		return state, err
	}
	if len(data) > updateMaxBytes || !utf8.Valid(data) {
		return state, ErrUnsafeTarget
	}
	// Detect a modification during the read, including a size change.
	var after unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &after); err != nil {
		return state, err
	}
	if st.Dev != after.Dev || st.Ino != after.Ino || st.Size != after.Size || st.Mtim != after.Mtim || st.Ctim != after.Ctim || int64(len(data)) != after.Size || st.Mode != after.Mode || st.Nlink != after.Nlink || st.Uid != after.Uid || st.Gid != after.Gid {
		return state, ErrUnsafeTarget
	}
	state = UpdateFileState{Content: string(data), SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), Bytes: len(data), Identity: fmt.Sprintf("%d:%d", st.Dev, st.Ino), Mode: st.Mode & 0o777, UID: st.Uid, GID: st.Gid, Links: uint64(st.Nlink)}
	// A readable but ineligible leaf can still supply an actual observation
	// on failure. Only a nil error establishes replacement eligibility.
	if st.Mode&0o7000 != 0 || st.Nlink != 1 {
		return state, ErrUnsafeTarget
	}
	if err := updateNoXattrs(file); err != nil {
		return state, err
	}
	if err := updateNoDefaultACL(parent); err != nil {
		return state, err
	}
	state.Metadata = updateMetadata
	return state, nil
}

func updateLeaf(parent *os.File, name string) (UpdateFileState, error) {
	file, err := openChild(parent, name)
	if err != nil {
		return UpdateFileState{}, err
	}
	defer func() { _ = file.Close() }()
	return updateState(file, parent)
}

func updateRead(root, relative string) (UpdateFileState, []ParentIdentity, error) {
	parts, err := updateParts(relative)
	if err != nil {
		return UpdateFileState{}, nil, err
	}
	dir, err := openRoot(root)
	if err != nil {
		return UpdateFileState{}, nil, err
	}
	defer func() { _ = dir.Close() }()
	parent, identities, opened, err := updateParent(dir, parts)
	defer updateClose(opened)
	if err != nil {
		return UpdateFileState{}, nil, err
	}
	state, err := updateLeaf(parent, parts[len(parts)-1])
	return state, identities, err
}

func updateSameParents(actual, expected []ParentIdentity) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i := range actual {
		if actual[i] != expected[i] {
			return false
		}
	}
	return true
}

// updateVisible checks the held root as well as the entire path currently
// visible from the root pathname. The held parent is separately checked by
// the caller before rename; this is not a compare-and-swap against writers.
func updateVisible(rootPath, relative, heldRoot, heldParent string, before UpdateFileState, ancestors []ParentIdentity) error {
	root, err := openRoot(rootPath)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	id, err := rootIdentity(root)
	if err != nil || id != heldRoot {
		return ErrUnsafeTarget
	}
	parts, _ := updateParts(relative)
	parent, actual, opened, err := updateParent(root, parts)
	defer updateClose(opened)
	if err != nil {
		return err
	}
	id, err = rootIdentity(parent)
	if err != nil || id != heldParent || !updateSameParents(actual, ancestors) {
		return ErrUnsafeTarget
	}
	state, err := updateLeaf(parent, parts[len(parts)-1])
	if err != nil {
		return err
	}
	if state != before {
		return ErrUnsafeTarget
	}
	return nil
}

func updateReplace(root, relative string, before UpdateFileState, after string, ancestors []ParentIdentity, recheck func() error) (result UpdateReplacement, err error) {
	result.Phase = "validate"
	parts, err := updateParts(relative)
	if err != nil {
		return result, err
	}
	if !utf8.ValidString(after) || len(after) > updateMaxBytes || recheck == nil {
		return result, ErrUnsafeTarget
	}
	result.Phase = "open"
	dir, err := openRoot(root)
	if err != nil {
		return result, err
	}
	defer func() { _ = dir.Close() }()
	rootID, err := rootIdentity(dir)
	if err != nil {
		return result, err
	}
	parent, actual, opened, err := updateParent(dir, parts)
	defer updateClose(opened)
	if err != nil {
		return result, err
	}
	parentID, err := rootIdentity(parent)
	if err != nil {
		return result, err
	}
	result.Phase = "recheck-before"
	if !updateSameParents(actual, ancestors) {
		return result, ErrUnsafeTarget
	}
	state, err := updateLeaf(parent, parts[len(parts)-1])
	if err != nil {
		return result, err
	}
	if state != before {
		return result, ErrUnsafeTarget
	}
	if err = updateVisible(root, relative, rootID, parentID, before, ancestors); err != nil {
		return result, err
	}
	result.Phase = "stage-create"
	var random [16]byte
	if _, err = rand.Read(random[:]); err != nil {
		return result, err
	}
	name := ".lds-update-" + hex.EncodeToString(random[:])
	fd, err := unix.Openat(int(parent.Fd()), name, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return result, err
	}
	result.Scratch = path.Join(path.Dir(relative), name)
	stage := os.NewFile(uintptr(fd), name)
	stageID, idErr := rootIdentity(stage)
	defer func() {
		if stage != nil {
			if closeErr := stage.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
		}
		if result.Scratch != "" {
			var observed unix.Stat_t
			checkErr := unix.Fstatat(int(parent.Fd()), name, &observed, unix.AT_SYMLINK_NOFOLLOW)
			if checkErr == nil && fmt.Sprintf("%d:%d", observed.Dev, observed.Ino) == stageID && observed.Mode&unix.S_IFMT == unix.S_IFREG {
				checkErr = unix.Unlinkat(int(parent.Fd()), name, 0)
			} else if checkErr == nil {
				checkErr = ErrUnsafeTarget
			}
			if checkErr == nil || errors.Is(checkErr, unix.ENOENT) {
				result.Scratch = ""
			} else {
				result.CleanupError = checkErr.Error()
			}
		}
	}()
	if idErr != nil {
		return result, idErr
	}
	result.Phase = "stage-write"
	var written int
	written, err = io.WriteString(stage, after)
	if err != nil {
		return result, err
	}
	if written != len(after) {
		return result, io.ErrShortWrite
	}
	result.Phase = "stage-verify"
	staged, err := updateState(stage, parent)
	if err != nil {
		return result, err
	}
	if staged.Content != after || staged.Bytes != len(after) || staged.UID != before.UID || staged.GID != before.GID {
		return result, ErrUnsafeTarget
	}
	result.Phase = "stage-mode"
	if err = stage.Chmod(os.FileMode(before.Mode)); err != nil {
		return result, err
	}
	staged, err = updateState(stage, parent)
	if err != nil {
		return result, err
	}
	if staged.Content != after || staged.Mode != before.Mode || staged.UID != before.UID || staged.GID != before.GID || staged.Identity != stageID {
		return result, ErrUnsafeTarget
	}
	result.Phase = "stage-sync"
	if err = stage.Sync(); err != nil {
		return result, err
	}
	if err = stage.Close(); err != nil {
		stage = nil
		return result, err
	}
	stage = nil
	result.Phase = "recheck-callback"
	if err = recheck(); err != nil {
		return result, err
	}
	result.Phase = "recheck-visible"
	if err = updateVisible(root, relative, rootID, parentID, before, ancestors); err != nil {
		return result, err
	}
	result.Phase = "stage-recheck"
	visibleStage, err := updateLeaf(parent, name)
	if err != nil {
		return result, err
	}
	if visibleStage != staged {
		return result, ErrUnsafeTarget
	}
	result.Phase = "rename"
	if err = unix.Renameat(int(parent.Fd()), name, int(parent.Fd()), parts[len(parts)-1]); err != nil {
		// A failed syscall is not proof that no replacement occurred. Preserve
		// the uncertainty and any after-state that can still be observed.
		result.Phase = "rename-uncertain"
		if visible, readErr := updateLeaf(parent, parts[len(parts)-1]); readErr == nil {
			result.Observed = &visible
			if visible.Identity == stageID {
				result.Renamed = true
				result.Scratch = ""
			}
		}
		return result, err
	}
	result.Renamed = true
	result.Scratch = ""
	result.Phase = "observe"
	observed, err := updateLeaf(parent, parts[len(parts)-1])
	if observed.Identity != "" {
		result.Observed = &observed
	}
	if err != nil {
		return result, err
	}
	if observed.Content != after || observed.Identity != stageID || observed.Mode != before.Mode || observed.UID != before.UID || observed.GID != before.GID || observed.Links != 1 || observed.Metadata != updateMetadata {
		return result, ErrUnsafeTarget
	}
	if err = updateVisible(root, relative, rootID, parentID, observed, ancestors); err != nil {
		return result, err
	}
	result.Phase = "parent-sync"
	if err = parent.Sync(); err != nil {
		return result, err
	}
	result.Phase = "complete"
	return result, nil
}
