//go:build !linux

package adoption

import "os"

// openRoot refuses on platforms without descriptor-relative no-follow opens.
func rootIdentity(*os.File) (string, error) { return "", ErrUnsafeTarget }
