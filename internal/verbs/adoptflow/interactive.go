package adoptflow

import (
	"bufio"
	"errors"
	"io"
	"os"
)

func isInteractive(input io.Reader, output io.Writer) bool {
	in, ok := input.(*os.File)
	if !ok || !isTerminalFile(in) {
		return false
	}
	out, ok := output.(*os.File)
	return ok && isTerminalFile(out)
}

func freshApprovalReader(input io.Reader) (*bufio.Reader, error) {
	file, ok := input.(*os.File)
	if !ok || !isTerminalFile(file) {
		return nil, errors.New("approval input is not a terminal file")
	}
	if err := flushTerminalInput(file); err != nil {
		return nil, err
	}
	return bufio.NewReader(file), nil
}
