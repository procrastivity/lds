//go:build !linux

package adoption

func createExclusive(*createTarget, string, []byte) error { return ErrUnsafeTarget }
