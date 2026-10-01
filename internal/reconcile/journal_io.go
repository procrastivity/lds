package reconcile

import (
	"crypto/rand"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

// Hold a no-follow directory handle before acquiring the journal lock. All
// subsequent journal I/O uses this handle, never a newly resolved parent path.
func openJournalDirectory(path string) (*os.File, error) {
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	dir := os.NewFile(uintptr(fd), "/")
	for _, part := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if part == "" {
			continue
		}
		nextFD, err := unix.Openat(int(dir.Fd()), part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		_ = dir.Close()
		if err != nil {
			return nil, err
		}
		dir = os.NewFile(uintptr(nextFD), part)
	}
	return dir, nil
}

func openJournalChild(dir *os.File, name string, flags int, mode uint32) (*os.File, error) {
	fd, err := unix.Openat(int(dir.Fd()), name, flags|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, mode)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}

func createJournalTemp(dir *os.File) (*os.File, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	return openJournalChild(dir, fmt.Sprintf(".lds-journal-%x.tmp", nonce), unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL, 0o600)
}
