package reconcile

// ReportWithObservation is the deterministic artifact form of ReportReviewed.
// Additional artifacts are read once, bounded and outside both project/Git
// trees; the original collector packet is rechecked rather than replaced.
func ReportWithObservation(input ReportInput, observedPath, draftPath, receiptPath string) (result ReportResult, resultErr error) {
	defer func() { resultErr = reportFailure(resultErr) }()
	current, err := CollectReport(input)
	if err != nil {
		return ReportResult{}, err
	}
	left, err := inspectReportRoot(input.LeftRoot, &projectFile{ProjectID: &current.Scan.Snapshots.Left.ProjectID, Revision: &current.Scan.Snapshots.Left.Revision})
	if err != nil {
		return ReportResult{}, err
	}
	right, err := inspectReportRoot(input.RightRoot, &projectFile{ProjectID: &current.Scan.Snapshots.Right.ProjectID, Revision: &current.Scan.Snapshots.Right.Revision})
	if err != nil {
		return ReportResult{}, err
	}
	for _, name := range []string{observedPath, draftPath, receiptPath} {
		if err := reportExternalPath(name, left, right); err != nil {
			return ReportResult{}, err
		}
	}
	data, _, err := readReportFile(observedPath, 8<<20)
	if err != nil {
		return ReportResult{}, err
	}
	var observed ReportEvidence
	if err := decodeStrictJSON(data, &observed); err != nil {
		return ReportResult{}, reportError("validation.reconcile-observation", err.Error())
	}
	if err := RecheckReport(input, observed); err != nil {
		return ReportResult{}, err
	}
	draft, _, err := readReportFile(draftPath, 64<<10)
	if err != nil {
		return ReportResult{}, err
	}
	data, _, err = readReportFile(receiptPath, 16<<10)
	if err != nil {
		return ReportResult{}, err
	}
	var receipt ReportAssistance
	if err := decodeStrictJSON(data, &receipt); err != nil {
		return ReportResult{}, reportError("validation.reconcile-receipt", err.Error())
	}
	return ReportReviewed(input, observed, draft, receipt)
}
