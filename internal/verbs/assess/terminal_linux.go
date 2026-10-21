//go:build linux

package assess

import (
	"bufio"
	"errors"
	"os"

	"golang.org/x/sys/unix"

	"github.com/procrastivity/lds/internal/iostreams"
)

func isInteractive(streams *iostreams.Streams) bool {
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

func freshConsent(streams *iostreams.Streams) (*bufio.Reader, error) {
	if !isInteractive(streams) {
		return nil, errors.New("interactive terminal unavailable")
	}
	file := streams.In.(*os.File)
	if err := unix.IoctlSetInt(int(file.Fd()), unix.TCFLSH, unix.TCIFLUSH); err != nil {
		return nil, err
	}
	return bufio.NewReader(file), nil
}
