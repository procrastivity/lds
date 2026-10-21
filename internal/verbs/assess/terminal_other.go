//go:build !linux

package assess

import (
	"bufio"
	"errors"

	"github.com/procrastivity/lds/internal/iostreams"
)

func isInteractive(*iostreams.Streams) bool { return false }

func freshConsent(*iostreams.Streams) (*bufio.Reader, error) {
	return nil, errors.New("Linux terminal required")
}
