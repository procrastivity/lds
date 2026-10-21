//go:build !linux

package iostreams

import (
	"bufio"
	"errors"
)

// IsInteractive refuses workflows requiring Linux terminal consent.
func (*Streams) IsInteractive() bool { return false }

// FreshInput refuses workflows requiring Linux terminal consent.
func (*Streams) FreshInput() (*bufio.Reader, error) {
	return nil, errors.New("Linux terminal required")
}
