//go:build linux

package reconcile

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

func openReportDirectory(name string) (*os.File, error) {
	if !filepath.IsAbs(name) || filepath.Clean(name) != name {
		return nil, reportError("validation.reconcile-report-path", "expected a clean absolute path")
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	for _, part := range strings.Split(strings.TrimPrefix(name, "/"), "/") {
		if part == "" {
			continue
		}
		next, err := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		_ = unix.Close(fd)
		if err != nil {
			return nil, err
		}
		fd = next
	}
	return os.NewFile(uintptr(fd), name), nil
}

func reportDirectoryIdentity(name string) (string, error) {
	dir, err := openReportDirectory(name)
	if err != nil {
		return "", err
	}
	defer func() { _ = dir.Close() }()
	var stat unix.Stat_t
	if err := unix.Fstat(int(dir.Fd()), &stat); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d:%d", stat.Dev, stat.Ino), nil
}

func readReportFile(name string, limit int) ([]byte, reportObservation, error) {
	observation := reportObservation{Path: name}
	if !filepath.IsAbs(name) || filepath.Clean(name) != name {
		return nil, observation, reportError("validation.reconcile-report-path", "expected a clean absolute file path")
	}
	parent, err := openReportDirectory(filepath.Dir(name))
	if err != nil {
		return nil, observation, err
	}
	defer func() { _ = parent.Close() }()
	fd, err := unix.Openat(int(parent.Fd()), filepath.Base(name), unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, observation, err
	}
	file := os.NewFile(uintptr(fd), name)
	defer func() { _ = file.Close() }()
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return nil, observation, err
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG {
		return nil, observation, reportError("validation.reconcile-report-path", "report input must be a regular file")
	}
	if stat.Size > int64(limit) {
		return nil, observation, reportError("validation.reconcile-report-budget", "report input exceeds the declared byte budget")
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return nil, observation, err
	}
	if len(data) > limit {
		return nil, observation, reportError("validation.reconcile-report-budget", "report input grew beyond the declared byte budget")
	}
	observation.SHA256, observation.Bytes = digest(data), len(data)
	observation.Identity = fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)
	return data, observation, nil
}

func checkReportOutput(name string) error {
	if !filepath.IsAbs(name) || filepath.Clean(name) != name {
		return reportError("validation.reconcile-report-path", "report output must be a clean absolute path")
	}
	parent, err := openReportDirectory(filepath.Dir(name))
	if err != nil {
		return err
	}
	defer func() { _ = parent.Close() }()
	var stat unix.Stat_t
	err = unix.Fstatat(int(parent.Fd()), filepath.Base(name), &stat, unix.AT_SYMLINK_NOFOLLOW)
	if err == nil {
		return reportError("refusal.reconcile-report-output", "report path is occupied; choose a new external path")
	}
	if !errors.Is(err, unix.ENOENT) {
		return err
	}
	return nil
}

func publishReport(name, expectedParent string, data []byte) (resultErr error) {
	parent, err := openReportDirectory(filepath.Dir(name))
	if err != nil {
		return err
	}
	defer func() { _ = parent.Close() }()
	var stat unix.Stat_t
	if err := unix.Fstat(int(parent.Fd()), &stat); err != nil {
		return err
	}
	if fmt.Sprintf("%d:%d", stat.Dev, stat.Ino) != expectedParent {
		return reportError("refusal.reconcile-report-stale", "report parent changed before publication")
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	temp := fmt.Sprintf(".lds-reconcile-report-%x.tmp", random)
	fd, err := unix.Openat(int(parent.Fd()), temp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), temp)
	defer func() { _ = file.Close() }()
	var owned unix.Stat_t
	if err := unix.Fstat(fd, &owned); err != nil {
		return err
	}
	defer func() {
		var current unix.Stat_t
		checkErr := unix.Fstatat(int(parent.Fd()), temp, &current, unix.AT_SYMLINK_NOFOLLOW)
		if checkErr == nil && current.Dev == owned.Dev && current.Ino == owned.Ino {
			checkErr = unix.Unlinkat(int(parent.Fd()), temp, 0)
		} else if checkErr == nil {
			checkErr = fmt.Errorf("scratch identity changed")
		}
		if checkErr != nil && !errors.Is(checkErr, unix.ENOENT) {
			resultErr = reportError("internal.reconcile-report-publication", fmt.Sprintf("report publication/cleanup uncertain at %s; inspect scratch %s: %v; %v", name, temp, resultErr, checkErr))
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		return err
	}
	if n, err := file.Write(data); err != nil || n != len(data) {
		if err == nil {
			err = io.ErrShortWrite
		}
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	// linkat publishes only a complete file and refuses any occupied leaf,
	// including a dangling symlink. It cannot overwrite an existing report.
	if err := unix.Linkat(int(parent.Fd()), temp, int(parent.Fd()), filepath.Base(name), 0); err != nil {
		if errors.Is(err, unix.EEXIST) {
			return reportError("refusal.reconcile-report-output", "report path became occupied; existing bytes were not replaced")
		}
		return err
	}
	if err := unix.Unlinkat(int(parent.Fd()), temp, 0); err != nil {
		return reportError("internal.reconcile-report-publication", "report was published but scratch cleanup failed; inspect "+name+": "+err.Error())
	}
	if err := parent.Sync(); err != nil {
		return reportError("internal.reconcile-report-publication", "report is visible but directory sync failed; inspect "+name+": "+err.Error())
	}
	return nil
}
