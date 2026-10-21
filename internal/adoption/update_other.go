//go:build !linux

package adoption

func updateRead(string, string) (UpdateFileState, []ParentIdentity, error) {
	return UpdateFileState{}, nil, ErrUnsafeTarget
}

func updateReplace(string, string, UpdateFileState, string, []ParentIdentity, func() error) (UpdateReplacement, error) {
	return UpdateReplacement{Phase: "platform"}, ErrUnsafeTarget
}
