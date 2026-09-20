package renderer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/smford/tf-blast/pkg/analyzer"
	"github.com/smford/tf-blast/pkg/graph"
)

func sampleReport() *analyzer.AnalysisReport {
	tree := &graph.TreeNode{
		Address: "aws_rds_cluster.primary",
		Action:  "delete,create",
		Children: []*graph.TreeNode{
			{
				Address: "module.services.aws_ecs_service.web",
				Action:  "",
			},
		},
	}

	return &analyzer.AnalysisReport{
		Summary: analyzer.Summary{
			ToAdd:            12,
			ToUpdate:         4,
			ToDestroy:        1,
			ToReplace:        3,
			TotalChanges:     20,
			TotalBlastRadius: 15,
			MaxSeverity:      analyzer.SeverityCritical,
			PlanHealth:       "CRITICAL BLAST RADIUS DETECTED",
		},
		Resources: []analyzer.ResourceAnalysis{
			{
				Address:            "aws_rds_cluster.primary",
				Type:               "aws_rds_cluster",
				Action:             analyzer.ActionReplace,
				Severity:           analyzer.SeverityCritical,
				RootCause:          "engine_version (downgrade)",
				DownstreamCount:    8,
				DownstreamAffected: []string{"module.services.aws_ecs_service.web"},
				DownstreamTree:     tree,
			},
			{
				Address:         "aws_security_group.db",
				Type:            "aws_security_group",
				Action:          analyzer.ActionDestroy,
				Severity:        analyzer.SeverityHigh,
				RootCause:       "Dropped in PR",
				DownstreamCount: 3,
			},
		},
	}
}

func TestRenderTerminal(t *testing.T) {
	report := sampleReport()
	var buf bytes.Buffer

	if err := RenderTerminal(&buf, report); err != nil {
		t.Fatalf("unexpected error rendering terminal: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Plan Health") {
		t.Errorf("missing Plan Health in output: %s", out)
	}
	if !strings.Contains(out, "aws_rds_cluster.primary") {
		t.Errorf("missing resource address in output: %s", out)
	}
	if !strings.Contains(out, "engine_version (downgrade)") {
		t.Errorf("missing root cause in output: %s", out)
	}
	if !strings.Contains(out, "Cascading Blast Radius Trees") {
		t.Errorf("missing cascading tree section in output: %s", out)
	}
}

func TestRenderMarkdown(t *testing.T) {
	report := sampleReport()
	var buf bytes.Buffer

	if err := RenderMarkdown(&buf, report); err != nil {
		t.Fatalf("unexpected error rendering markdown: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "### 💥 tf-blast: Blast-Radius & Risk Report") {
		t.Errorf("missing header in markdown: %s", out)
	}
	if !strings.Contains(out, "| 🔴 CRITICAL | `aws_rds_cluster.primary` | **REPLACE** | `engine_version (downgrade)` | 8 services |") {
		t.Errorf("missing critical table row in markdown: %s", out)
	}
	if !strings.Contains(out, "<details>") || !strings.Contains(out, "<summary>🔍 <strong>View Cascading Blast Radius Graph</strong></summary>") {
		t.Errorf("missing collapsible details in markdown: %s", out)
	}
}

func TestRenderJSON(t *testing.T) {
	report := sampleReport()
	var buf bytes.Buffer

	if err := RenderJSON(&buf, report); err != nil {
		t.Fatalf("unexpected error rendering JSON: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"to_replace": 3`) {
		t.Errorf("missing to_replace metric in JSON: %s", out)
	}
	if !strings.Contains(out, `"aws_rds_cluster.primary"`) {
		t.Errorf("missing resource address in JSON: %s", out)
	}
}

func TestRenderMermaid(t *testing.T) {
	report := sampleReport()
	var buf bytes.Buffer

	if err := RenderMermaid(&buf, report); err != nil {
		t.Fatalf("unexpected error rendering Mermaid: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "```mermaid") || !strings.Contains(out, "graph TD") {
		t.Errorf("missing mermaid fence in output: %s", out)
	}
	if !strings.Contains(out, "aws_rds_cluster.primary") {
		t.Errorf("missing rds cluster in mermaid diagram: %s", out)
	}
	if !strings.Contains(out, "module.services.aws_ecs_service.web") {
		t.Errorf("missing ecs service in mermaid diagram: %s", out)
	}
}

func TestRenderSARIF(t *testing.T) {
	report := sampleReport()
	var buf bytes.Buffer

	if err := RenderSARIF(&buf, report, "1.0.0"); err != nil {
		t.Fatalf("unexpected error rendering SARIF: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"version": "2.1.0"`) {
		t.Errorf("missing SARIF version: %s", out)
	}
	if !strings.Contains(out, "TF-BLAST-001") {
		t.Errorf("missing critical rule TF-BLAST-001 in SARIF: %s", out)
	}
	if !strings.Contains(out, "aws_rds_cluster.primary") {
		t.Errorf("missing rds cluster in SARIF results: %s", out)
	}
}

func TestRenderHTML(t *testing.T) {
	report := sampleReport()
	var buf bytes.Buffer

	if err := RenderHTML(&buf, report); err != nil {
		t.Fatalf("unexpected error rendering HTML: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Errorf("missing DOCTYPE in HTML output")
	}
	if !strings.Contains(out, "aws_rds_cluster.primary") {
		t.Errorf("missing rds cluster in HTML output")
	}
	if !strings.Contains(out, "Blast Score") {
		t.Errorf("missing Blast Score in HTML output")
	}
}
