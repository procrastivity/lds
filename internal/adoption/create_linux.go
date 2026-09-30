//go:build linux

package adoption

import (
	"errors"
	"os"
	"path"
	"strings"
	"syscall"
)

func createExclusive(target *createTarget, relative string, data []byte) error {
	if path.Clean(relative) != relative || strings.HasPrefix(relative, "/") {
		return ErrUnsafeTarget
	}
	components := strings.Split(relative, "/")
	if len(components) < 2 {
		return ErrUnsafeTarget
	}
	if err := checkVisibleRoot(target); err != nil {
		return err
	}
	parent := target.root
	opened := []*os.File{}
	defer func() {
		for _, file := range opened {
			_ = file.Close()
		}
	}()
	for i, component := range components[:len(components)-1] {
		parentPath := strings.Join(components[:i+1], "/") + "/"
		next, err := openChild(parent, component)
		created := false
		if errors.Is(err, os.ErrNotExist) {
			if !target.missingParents[parentPath] || target.createdParents[parentPath] != "" {
				return ErrUnsafeTarget
			}
			if err := syscall.Mkdirat(int(parent.Fd()), component, 0o755); err != nil {
				return err
			}
			next, err = openChild(parent, component)
			created = err == nil
		}
		if err != nil {
			return err
		}
		info, err := next.Stat()
		if err != nil {
			_ = next.Close()
			return err
		}
		if !info.IsDir() {
			_ = next.Close()
			return ErrUnsafeTarget
		}
		identity, err := rootIdentity(next)
		if err != nil {
			_ = next.Close()
			return err
		}
		if created {
			target.createdParents[parentPath] = identity
		} else if expected, ok := target.existingParents[parentPath]; ok {
			if identity != expected {
				_ = next.Close()
				return ErrUnsafeTarget
			}
		} else if expected, ok := target.createdParents[parentPath]; ok {
			if identity != expected {
				_ = next.Close()
				return ErrUnsafeTarget
			}
		} else if target.missingParents[parentPath] {
			// A parent newly observed after review is valid only if this call
			// created it; an independent new directory is stale input.
			_ = next.Close()
			return ErrUnsafeTarget
		} else {
			_ = next.Close()
			return ErrUnsafeTarget
		}
		opened = append(opened, next)
		parent = next
	}
	if err := checkVisibleParents(target, components[:len(components)-1]); err != nil {
		return err
	}
	fd, err := syscall.Openat(int(parent.Fd()), components[len(components)-1], syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0o644)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), components[len(components)-1])
	written, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if written != len(data) {
		return syscall.EIO
	}
	return closeErr
}

func checkVisibleRoot(target *createTarget) error {
	visible, err := openRoot(target.rootPath)
	if err != nil {
		return ErrUnsafeTarget
	}
	defer func() { _ = visible.Close() }()
	identity, err := rootIdentity(visible)
	if err != nil || identity != target.rootIdentity {
		return ErrUnsafeTarget
	}
	return nil
}

func checkVisibleParents(target *createTarget, components []string) error {
	visible, err := openRoot(target.rootPath)
	if err != nil {
		return ErrUnsafeTarget
	}
	defer func() { _ = visible.Close() }()
	identity, err := rootIdentity(visible)
	if err != nil || identity != target.rootIdentity {
		return ErrUnsafeTarget
	}
	parent := visible
	for i, component := range components {
		next, err := openChild(parent, component)
		if parent != visible {
			_ = parent.Close()
		}
		if err != nil {
			return ErrUnsafeTarget
		}
		parent = next
		identity, err := rootIdentity(next)
		if err != nil {
			_ = next.Close()
			return ErrUnsafeTarget
		}
		name := strings.Join(components[:i+1], "/") + "/"
		expected := target.existingParents[name]
		if expected == "" {
			expected = target.createdParents[name]
		}
		if expected == "" || identity != expected {
			_ = next.Close()
			return ErrUnsafeTarget
		}
	}
	if parent != visible {
		_ = parent.Close()
	}
	return nil
}
