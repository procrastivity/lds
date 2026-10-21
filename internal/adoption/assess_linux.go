//go:build linux

package adoption

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/procrastivity/lds/internal/ldserr"
)

func writeAssessmentReport(name, targetIdentity string, data []byte) error {
	parent, err := openRoot(filepath.Dir(name))
	if err != nil {
		return err
	}
	defer func() { _ = parent.Close() }()
	identity, err := rootIdentity(parent)
	if err != nil {
		return err
	}
	if identity == targetIdentity {
		return ldserr.New("refusal.assessment-output", "report parent is the target directory")
	}
	visible, err := openRoot(filepath.Dir(name))
	if err != nil {
		return err
	}
	visibleIdentity, err := rootIdentity(visible)
	_ = visible.Close()
	if err != nil || visibleIdentity != identity {
		return ldserr.New("refusal.assessment-stale", "report parent changed before creation")
	}
	fd, err := syscall.Openat(int(parent.Fd()), filepath.Base(name), syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0o600)
	if errors.Is(err, os.ErrExist) {
		return ldserr.New("refusal.assessment-output", "report path is occupied; choose a new path, existing bytes were not replaced")
	}
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), name)
	written, writeErr := file.Write(data)
	closeErr := file.Close()
	if written != len(data) && writeErr == nil {
		writeErr = io.ErrShortWrite
	}
	if err := errors.Join(writeErr, closeErr); err != nil {
		return ldserr.New("internal.assessment-report", "report write failed and may be partial; inspect the external report before continuing: "+err.Error())
	}
	return nil
}
