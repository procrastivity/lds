//go:build !linux

package adoption

import "os"

// Without no-follow descriptor-relative opens, refuse instead of silently
// weakening the target boundary on an unsupported platform.
func openRoot(string) (*os.File, error)            { return nil, ErrUnsafeTarget }
func openChild(*os.File, string) (*os.File, error) { return nil, ErrUnsafeTarget }
