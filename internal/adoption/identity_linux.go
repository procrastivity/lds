//go:build linux

package adoption

import (
	"fmt"
	"os"
	"syscall"
)

// The identity comes from the held no-follow descriptor, not a second path
// lookup that could observe a replacement directory.
func rootIdentity(root *os.File) (string, error) {
	info, err := root.Stat()
	if err != nil {
		return "", err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", ErrUnsafeTarget
	}
	return fmt.Sprintf("%d:%d", stat.Dev, stat.Ino), nil
}
