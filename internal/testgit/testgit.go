// Package testgit isolates Git configuration for fixture-based test suites.
package testgit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Run executes a suite with a Git shim that ignores system and user config,
// while preserving repository-local config and normal Git behavior. The shim
// sets Git's environment after the scanner strips inherited GIT_* variables;
// setting those variables only in TestMain would not isolate scanner reads.
func Run(m *testing.M) int {
	shell, err := exec.LookPath("sh")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	shell, err = filepath.Abs(shell)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	git, err := exec.LookPath("git")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	git, err = filepath.Abs(git)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	dir, err := os.MkdirTemp("", "lds-test-git-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = os.RemoveAll(dir) }()

	quotedGit := "'" + strings.ReplaceAll(git, "'", "'\\''") + "'"
	script := "#!" + shell + "\n" +
		"export GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null\n" +
		"exec " + quotedGit + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o700); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	path := os.Getenv("PATH")
	if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = os.Setenv("PATH", path) }()
	return m.Run()
}
