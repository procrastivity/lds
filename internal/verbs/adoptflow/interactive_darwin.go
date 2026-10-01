//go:build darwin

package adoptflow

import (
	"os"

	"golang.org/x/sys/unix"
)

func isTerminalFile(file *os.File) bool {
	_, err := unix.IoctlGetTermios(int(file.Fd()), unix.TIOCGETA)
	return err == nil
}

func flushTerminalInput(file *os.File) error {
	return unix.IoctlSetPointerInt(int(file.Fd()), unix.TIOCFLUSH, unix.TCIFLUSH)
}
