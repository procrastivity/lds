//go:build !linux && !darwin

package adoptflow

import (
	"errors"
	"os"
)

func isTerminalFile(*os.File) bool { return false }

func flushTerminalInput(*os.File) error { return errors.New("terminal input flush is unsupported") }
