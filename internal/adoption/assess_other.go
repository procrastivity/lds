//go:build !linux

package adoption

func writeAssessmentReport(string, string, []byte) error { return ErrUnsafeTarget }
