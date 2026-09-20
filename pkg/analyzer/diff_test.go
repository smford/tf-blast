package analyzer

import (
	"testing"
)

func TestDiffReports(t *testing.T) {
	before := &AnalysisReport{
		Summary: Summary{
			TotalBlastRadius: 10,
			BlastScore:       45,
			TotalChanges:     5,
		},
		Resources: []ResourceAnalysis{
			{
				Address:   "aws_rds_cluster.db",
				Severity:  SeverityCritical,
				Action:    ActionReplace,
				RiskScore: 30,
			},
			{
				Address:   "aws_instance.app",
				Severity:  SeverityLow,
				Action:    ActionUpdateInPlace,
				RiskScore: 1,
			},
		},
	}

	after := &AnalysisReport{
		Summary: Summary{
			TotalBlastRadius: 3,
			BlastScore:       10,
			TotalChanges:     2,
		},
		Resources: []ResourceAnalysis{
			{
				Address:   "aws_instance.app",
				Severity:  SeverityLow,
				Action:    ActionUpdateInPlace,
				RiskScore: 1,
			},
			{
				Address:   "aws_s3_bucket.data",
				Severity:  SeverityLow,
				Action:    ActionCreate,
				RiskScore: 1,
			},
		},
	}

	diff := DiffReports(before, after)

	if diff.RadiusDelta != -7 {
		t.Errorf("expected RadiusDelta -7, got %d", diff.RadiusDelta)
	}
	if diff.ScoreDelta != -35 {
		t.Errorf("expected ScoreDelta -35, got %d", diff.ScoreDelta)
	}
	if len(diff.ResolvedRisks) != 1 || diff.ResolvedRisks[0].Address != "aws_rds_cluster.db" {
		t.Errorf("expected aws_rds_cluster.db in ResolvedRisks")
	}
	if len(diff.NewRisks) != 1 || diff.NewRisks[0].Address != "aws_s3_bucket.data" {
		t.Errorf("expected aws_s3_bucket.data in NewRisks")
	}
}
