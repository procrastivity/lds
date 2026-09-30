//go:build linux

package adoption

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Open every component relative to an already-open directory. O_NOFOLLOW
// applies at the actual open, so a symlink swap cannot redirect a later read.
func openRoot(path string) (*os.File, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dir, err := os.Open("/")
	if err != nil {
		return nil, err
	}
	for _, part := range strings.Split(strings.TrimPrefix(abs, "/"), "/") {
		if part == "" {
			continue
		}
		next, err := openChild(dir, part)
		_ = dir.Close()
		if err != nil {
			return nil, err
		}
		info, err := next.Stat()
		if err != nil {
			_ = next.Close()
			return nil, err
		}
		if !info.IsDir() {
			_ = next.Close()
			return nil, fmt.Errorf("%w: root ancestor is not a directory", ErrUnsafeTarget)
		}
		dir = next
	}
	return dir, nil
}

func openChild(parent *os.File, name string) (*os.File, error) {
	fd, err := syscall.Openat(int(parent.Fd()), name, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC|syscall.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ELOOP) {
		return nil, fmt.Errorf("%w: symlink at %s", ErrUnsafeTarget, name)
	}
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}
