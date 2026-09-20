package analyzer

// DiffReport captures the comparative blast radius and risk changes between two execution plans.
type DiffReport struct {
	BeforeSummary Summary            `json:"before_summary"`
	AfterSummary  Summary            `json:"after_summary"`
	RadiusDelta   int                `json:"radius_delta"`
	ScoreDelta    int                `json:"score_delta"`
	ChangesDelta  int                `json:"changes_delta"`
	ResolvedRisks []ResourceAnalysis `json:"resolved_risks"`
	NewRisks      []ResourceAnalysis `json:"new_risks"`
	ModifiedRisks []RiskDelta        `json:"modified_risks"`
}

// RiskDelta details how a specific resource's risk profile evolved between two plans.
type RiskDelta struct {
	Address        string     `json:"address"`
	BeforeSeverity Severity   `json:"before_severity"`
	AfterSeverity  Severity   `json:"after_severity"`
	BeforeAction   ActionType `json:"before_action"`
	AfterAction    ActionType `json:"after_action"`
	BeforeScore    int        `json:"before_score"`
	AfterScore     int        `json:"after_score"`
}

// DiffReports compares a previous AnalysisReport against a subsequent AnalysisReport.
func DiffReports(before, after *AnalysisReport) *DiffReport {
	diff := &DiffReport{
		ResolvedRisks: make([]ResourceAnalysis, 0),
		NewRisks:      make([]ResourceAnalysis, 0),
		ModifiedRisks: make([]RiskDelta, 0),
	}

	if before != nil {
		diff.BeforeSummary = before.Summary
	}
	if after != nil {
		diff.AfterSummary = after.Summary
	}

	diff.RadiusDelta = diff.AfterSummary.TotalBlastRadius - diff.BeforeSummary.TotalBlastRadius
	diff.ScoreDelta = diff.AfterSummary.BlastScore - diff.BeforeSummary.BlastScore
	diff.ChangesDelta = diff.AfterSummary.TotalChanges - diff.BeforeSummary.TotalChanges

	beforeMap := make(map[string]ResourceAnalysis)
	if before != nil {
		for _, r := range before.Resources {
			beforeMap[r.Address] = r
		}
	}

	afterMap := make(map[string]ResourceAnalysis)
	if after != nil {
		for _, r := range after.Resources {
			afterMap[r.Address] = r
		}
	}

	// 1. Identify New Risks & Modified Risks
	for addr, afterRes := range afterMap {
		beforeRes, exists := beforeMap[addr]
		if !exists {
			diff.NewRisks = append(diff.NewRisks, afterRes)
		} else {
			if beforeRes.Severity != afterRes.Severity ||
				beforeRes.Action != afterRes.Action ||
				beforeRes.RiskScore != afterRes.RiskScore {
				diff.ModifiedRisks = append(diff.ModifiedRisks, RiskDelta{
					Address:        addr,
					BeforeSeverity: beforeRes.Severity,
					AfterSeverity:  afterRes.Severity,
					BeforeAction:   beforeRes.Action,
					AfterAction:    afterRes.Action,
					BeforeScore:    beforeRes.RiskScore,
					AfterScore:     afterRes.RiskScore,
				})
			}
		}
	}

	// 2. Identify Resolved / Mitigated Risks (present in before, but absent or noop in after)
	for addr, beforeRes := range beforeMap {
		if _, exists := afterMap[addr]; !exists {
			diff.ResolvedRisks = append(diff.ResolvedRisks, beforeRes)
		}
	}

	return diff
}
