//go:build !linux

package reconcile

func reportUnsupported() error {
	return reportError("refusal.reconcile-report-platform", "offline reconciliation reports require Linux; no report or source/journal write was made")
}

func reportDirectoryIdentity(string) (string, error) { return "", reportUnsupported() }
func readReportFile(string, int) ([]byte, reportObservation, error) {
	return nil, reportObservation{}, reportUnsupported()
}
func checkReportOutput(string) error             { return reportUnsupported() }
func publishReport(string, string, []byte) error { return reportUnsupported() }
