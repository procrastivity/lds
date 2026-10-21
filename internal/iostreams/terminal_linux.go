//go:build linux

package iostreams

import (
	"bufio"
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// IsInteractive reports whether input and presentation use Linux terminals.
func (streams *Streams) IsInteractive() bool {
	in, ok := streams.In.(*os.File)
	if !ok {
		return false
	}
	out, ok := streams.Err.(*os.File)
	if !ok {
		return false
	}
	_, inErr := unix.IoctlGetTermios(int(in.Fd()), unix.TCGETS)
	_, outErr := unix.IoctlGetTermios(int(out.Fd()), unix.TCGETS)
	return inErr == nil && outErr == nil
}

// FreshInput discards queued terminal input after presenting a review, then
// returns a new reader. Input from before this boundary cannot grant consent.
func (streams *Streams) FreshInput() (*bufio.Reader, error) {
	if !streams.IsInteractive() {
		return nil, errors.New("interactive terminal unavailable")
	}
	file := streams.In.(*os.File)
	if err := unix.IoctlSetInt(int(file.Fd()), unix.TCFLSH, unix.TCIFLUSH); err != nil {
		return nil, err
	}
	return bufio.NewReader(file), nil
}
