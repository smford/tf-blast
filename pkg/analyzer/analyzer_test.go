package analyzer

import (
	"testing"

	"github.com/smford/tf-blast/pkg/config"
	"github.com/smford/tf-blast/pkg/graph"
	"github.com/smford/tf-blast/pkg/parser"
)

func TestAnalyze_RDSReplacement(t *testing.T) {
	plan := &parser.Plan{
		ResourceChanges: []parser.ResourceChange{
			{
				Address: "aws_rds_cluster.primary",
				Type:    "aws_rds_cluster",
				Name:    "primary",
				Change: parser.Change{
					Actions:      []string{"delete", "create"},
					Before:       map[string]any{"engine_version": "14.3"},
					After:        map[string]any{"engine_version": "13.7"},
					ReplacePaths: [][]any{{"engine_version"}},
				},
			},
		},
	}

	g := graph.NewGraph()
	// Add 8 downstream services
	for i := 1; i <= 8; i++ {
		serviceAddr := "module.services.aws_ecs_service.service_" + string(rune('0'+i))
		g.AddEdge(serviceAddr, "aws_rds_cluster.primary")
	}

	cfg := config.DefaultConfig()
	report := Analyze(plan, g, cfg)

	if report.Summary.MaxSeverity != SeverityCritical {
		t.Errorf("expected CRITICAL severity for RDS replacement, got %s", report.Summary.MaxSeverity)
	}
	if report.Summary.ToReplace != 1 {
		t.Errorf("expected 1 replacement, got %d", report.Summary.ToReplace)
	}
	if report.Summary.TotalBlastRadius != 9 { // 1 RDS + 8 ECS services
		t.Errorf("expected total blast radius 9, got %d", report.Summary.TotalBlastRadius)
	}
	if len(report.Resources) != 1 {
		t.Fatalf("expected 1 analyzed resource, got %d", len(report.Resources))
	}

	res := report.Resources[0]
	if res.RootCause != "engine_version (downgrade)" {
		t.Errorf("expected root cause 'engine_version (downgrade)', got '%s'", res.RootCause)
	}
	if res.DownstreamCount != 8 {
		t.Errorf("expected 8 downstream affected, got %d", res.DownstreamCount)
	}
}

func TestAnalyze_FailOnThresholds(t *testing.T) {
	plan := &parser.Plan{
		ResourceChanges: []parser.ResourceChange{
			{
				Address: "aws_security_group.db",
				Type:    "aws_security_group",
				Name:    "db",
				Change: parser.Change{
					Actions: []string{"delete"},
				},
			},
		},
	}

	cfg := config.DefaultConfig()
	cfg.FailOn = "high"

	g := graph.NewGraph()
	report := Analyze(plan, g, cfg)

	if !report.Failed {
		t.Errorf("expected report to fail on 'high' threshold for security group deletion")
	}

	cfg.FailOn = "critical"
	reportCrit := Analyze(plan, g, cfg)
	if reportCrit.Failed {
		t.Errorf("did not expect report to fail on 'critical' threshold for isolated security group deletion")
	}

	cfg.FailOn = "any-destroy"
	reportDestroy := Analyze(plan, g, cfg)
	if !reportDestroy.Failed {
		t.Errorf("expected report to fail on 'any-destroy'")
	}
}

func TestAnalyze_MaxBlastThreshold(t *testing.T) {
	plan := &parser.Plan{
		ResourceChanges: []parser.ResourceChange{
			{
				Address: "aws_instance.app",
				Type:    "aws_instance",
				Name:    "app",
				Change: parser.Change{
					Actions: []string{"delete"},
				},
			},
		},
	}

	g := graph.NewGraph()
	// Add 5 downstream resources
	for i := 1; i <= 5; i++ {
		g.AddEdge("aws_route53_record.rec_"+string(rune('0'+i)), "aws_instance.app")
	}

	cfg := config.DefaultConfig()
	cfg.MaxBlast = 3 // Exceeded because total blast is 6 (1 instance + 5 records)

	report := Analyze(plan, g, cfg)
	if !report.Failed {
		t.Errorf("expected report to fail because blast radius (6) > max_blast (3)")
	}
}

func TestAnalyze_SensitiveAttributeRedaction(t *testing.T) {
	plan := &parser.Plan{
		ResourceChanges: []parser.ResourceChange{
			{
				Address: "aws_db_instance.db",
				Type:    "aws_db_instance",
				Name:    "db",
				Change: parser.Change{
					Actions:         []string{"update"},
					Before:          map[string]any{"master_password": "supersecretpassword123"},
					After:           map[string]any{"master_password": "newsupersecretpassword456"},
					BeforeSensitive: map[string]any{"master_password": true},
					AfterSensitive:  map[string]any{"master_password": true},
				},
			},
		},
	}

	report := Analyze(plan, nil, config.DefaultConfig())
	if len(report.Resources) != 1 {
		t.Fatalf("expected 1 analyzed resource")
	}

	res := report.Resources[0]
	if !res.IsSensitive {
		t.Errorf("expected IsSensitive to be true")
	}
	if res.RootCause == "supersecretpassword123" || res.RootCause == "newsupersecretpassword456" {
		t.Errorf("leak detected! sensitive password value was not redacted: %s", res.RootCause)
	}
}

func TestAnalyze_ResourceDriftDetection(t *testing.T) {
	plan := &parser.Plan{
		ResourceChanges: []parser.ResourceChange{
			{
				Address: "aws_security_group.web",
				Type:    "aws_security_group",
				Name:    "web",
				Change: parser.Change{
					Actions: []string{"update"},
					Before:  map[string]any{"description": "web sg"},
					After:   map[string]any{"description": "web sg updated"},
				},
			},
		},
		ResourceDrift: []parser.ResourceDrift{
			{
				Address: "aws_security_group.web",
				Type:    "aws_security_group",
				Name:    "web",
				Change: parser.Change{
					Actions: []string{"update"},
				},
			},
		},
	}

	report := Analyze(plan, nil, config.DefaultConfig())
	if report.Summary.DriftCount != 1 {
		t.Errorf("expected 1 drifted resource, got %d", report.Summary.DriftCount)
	}
	if len(report.Resources) != 1 || !report.Resources[0].HasDrift {
		t.Errorf("expected resource to have HasDrift=true")
	}
}

func TestAnalyze_DisruptiveInPlaceResize(t *testing.T) {
	plan := &parser.Plan{
		ResourceChanges: []parser.ResourceChange{
			{
				Address: "aws_instance.worker",
				Type:    "aws_instance",
				Name:    "worker",
				Change: parser.Change{
					Actions: []string{"update"},
					Before:  map[string]any{"instance_type": "t3.large"},
					After:   map[string]any{"instance_type": "t3.nano"},
				},
			},
		},
	}

	report := Analyze(plan, nil, config.DefaultConfig())
	if len(report.Resources) != 1 {
		t.Fatalf("expected 1 analyzed resource")
	}

	res := report.Resources[0]
	// Disruptive instance resizing should be evaluated as MEDIUM risk rather than LOW
	if res.Severity != SeverityMedium {
		t.Errorf("expected SeverityMedium for instance_type resize, got %s", res.Severity)
	}
}

func TestAnalyze_MaxScoreThreshold(t *testing.T) {
	plan := &parser.Plan{
		ResourceChanges: []parser.ResourceChange{
			{
				Address: "aws_rds_cluster.db",
				Type:    "aws_rds_cluster",
				Change: parser.Change{
					Actions: []string{"delete"},
				},
			},
		},
	}

	// Critical deletion has risk score: 25 (critical) + 5 (destroy) = 30
	cfg := config.DefaultConfig()
	cfg.MaxScore = 20
	report := Analyze(plan, nil, cfg)

	if !report.Failed {
		t.Fatalf("expected report to fail when BlastScore (%d) exceeds MaxScore (20)", report.Summary.BlastScore)
	}
	if report.Summary.BlastScore < 30 {
		t.Errorf("expected BlastScore >= 30, got %d", report.Summary.BlastScore)
	}

	cfg.MaxScore = 50
	reportPass := Analyze(plan, nil, cfg)
	if reportPass.Failed {
		t.Fatalf("expected report to pass when BlastScore is under MaxScore 50")
	}
}
