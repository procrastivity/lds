//go:build linux

package adoptflow

import (
	"os"

	"golang.org/x/sys/unix"
)

func isTerminalFile(file *os.File) bool {
	_, err := unix.IoctlGetTermios(int(file.Fd()), unix.TCGETS)
	return err == nil
}

func flushTerminalInput(file *os.File) error {
	return unix.IoctlSetInt(int(file.Fd()), unix.TCFLSH, unix.TCIFLUSH)
}
