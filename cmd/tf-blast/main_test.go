package main_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build binary once for CLI integration tests
	tmpDir, err := os.MkdirTemp("", "tf-blast-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	bin := filepath.Join(tmpDir, "tf-blast")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	binaryPath = bin

	os.Exit(m.Run())
}

func TestCLI_CleanPlan(t *testing.T) {
	cmd := exec.Command(binaryPath, filepath.Join("..", "..", "testdata", "clean-plan.json"))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v, stderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "Plan Health: LOW RISK / HEALTHY") {
		t.Errorf("expected clean health status in output: %s", out)
	}
}

func TestCLI_ReplacementPlan_Default(t *testing.T) {
	cmd := exec.Command(binaryPath, filepath.Join("..", "..", "testdata", "replacement-plan.json"))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("expected exit 0 by default, got: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "aws_security_group.db") || !strings.Contains(out, "REPLACE") {
		t.Errorf("expected security group replace in output: %s", out)
	}
}

func TestCLI_ReplacementPlan_FailOnReplacement(t *testing.T) {
	cmd := exec.Command(binaryPath, "--fail-on", "replacement", filepath.Join("..", "..", "testdata", "replacement-plan.json"))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected exit code 1 with --fail-on replacement, but got exit 0")
	}

	out := stdout.String()
	if !strings.Contains(out, "Policy Check FAILED") {
		t.Errorf("expected policy failure notice in output: %s", out)
	}
}

func TestCLI_StatefulDestroy_FailOnCritical(t *testing.T) {
	cmd := exec.Command(binaryPath, "--fail-on", "critical", filepath.Join("..", "..", "testdata", "stateful-destroy.json"))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected exit code 1 for critical destroy, but got exit 0")
	}

	out := stdout.String()
	if !strings.Contains(out, "CRITICAL BLAST RADIUS DETECTED") {
		t.Errorf("expected CRITICAL health banner: %s", out)
	}
	if !strings.Contains(out, "aws_rds_cluster.primary") {
		t.Errorf("expected rds cluster address in output: %s", out)
	}
}

func TestCLI_StdinPiping(t *testing.T) {
	planData, err := os.ReadFile(filepath.Join("..", "..", "testdata", "clean-plan.json"))
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath)
	cmd.Stdin = bytes.NewReader(planData)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("piping via stdin failed: %v, stderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Plan Health") {
		t.Errorf("unexpected output from piped input: %s", stdout.String())
	}
}

func TestCLI_JSONOutput(t *testing.T) {
	cmd := exec.Command(binaryPath, "--json", filepath.Join("..", "..", "testdata", "clean-plan.json"))
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to run with --json: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	summary, ok := parsed["summary"].(map[string]any)
	if !ok {
		t.Fatalf("missing summary in JSON output")
	}

	if summary["to_add"].(float64) != 3 {
		t.Errorf("expected 3 to_add in json summary, got %v", summary["to_add"])
	}
}

func TestCLI_OutFileMarkdown(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "pr-comment.md")
	cmd := exec.Command(binaryPath, "--out-file", tmpFile, filepath.Join("..", "..", "testdata", "replacement-plan.json"))

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to run with --out-file: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read created out-file: %v", err)
	}

	strContent := string(content)
	if !strings.Contains(strContent, "### 💥 tf-blast: Blast-Radius & Risk Report") {
		t.Errorf("missing expected markdown header in file: %s", strContent)
	}
	if !strings.Contains(strContent, "| 🟠 HIGH | `aws_security_group.db` |") {
		t.Errorf("missing security group row in markdown file: %s", strContent)
	}
}

func TestCLI_MermaidOutput(t *testing.T) {
	cmd := exec.Command(binaryPath, "--output", "mermaid", filepath.Join("..", "..", "testdata", "replacement-plan.json"))
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to run with --output mermaid: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "```mermaid") || !strings.Contains(out, "graph TD") {
		t.Errorf("expected mermaid graph definition, got: %s", out)
	}
	if !strings.Contains(out, "n_aws_security_group_db") {
		t.Errorf("expected security group node in mermaid graph, got: %s", out)
	}
}

func TestCLI_SARIFOutput(t *testing.T) {
	cmd := exec.Command(binaryPath, "--output", "sarif", filepath.Join("..", "..", "testdata", "stateful-destroy.json"))
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to run with --output sarif: %v", err)
	}

	var sarif map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &sarif); err != nil {
		t.Fatalf("failed to parse SARIF output as JSON: %v", err)
	}

	if sarif["version"] != "2.1.0" {
		t.Errorf("expected SARIF version 2.1.0, got %v", sarif["version"])
	}
	runs, ok := sarif["runs"].([]any)
	if !ok || len(runs) == 0 {
		t.Fatalf("expected runs array in SARIF")
	}
}

func TestCLI_MultiPlanAggregation(t *testing.T) {
	plan1 := filepath.Join("..", "..", "testdata", "clean-plan.json")
	plan2 := filepath.Join("..", "..", "testdata", "replacement-plan.json")

	cmd := exec.Command(binaryPath, "--json", plan1, plan2)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to run multi-plan aggregation: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse aggregated JSON: %v", err)
	}

	summary := parsed["summary"].(map[string]any)
	// clean-plan has 3 creates, replacement-plan has 1 replace + 3 updates
	toAdd := summary["to_add"].(float64)
	toReplace := summary["to_replace"].(float64)
	if toAdd != 3 {
		t.Errorf("expected 3 to_add in multi-plan, got %v", toAdd)
	}
	if toReplace != 1 {
		t.Errorf("expected 1 to_replace in multi-plan, got %v", toReplace)
	}

	resources := parsed["resources"].([]any)
	// Check that plan sources or qualified addresses exist
	foundPlan1 := false
	foundPlan2 := false
	for _, r := range resources {
		rm := r.(map[string]any)
		ps := rm["plan_source"].(string)
		if strings.Contains(ps, "clean-plan") {
			foundPlan1 = true
		}
		if strings.Contains(ps, "replacement-plan") {
			foundPlan2 = true
		}
	}
	if !foundPlan1 || !foundPlan2 {
		t.Errorf("expected resources from both plans with plan_source annotated, foundPlan1=%v foundPlan2=%v", foundPlan1, foundPlan2)
	}
}

func TestCLI_HTMLOutput(t *testing.T) {
	cmd := exec.Command(binaryPath, "--output", "html", filepath.Join("..", "..", "testdata", "clean-plan.json"))
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to run with --output html: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Errorf("expected HTML output, got: %s", out)
	}
	if !strings.Contains(out, "tf-blast") {
		t.Errorf("expected tf-blast header in HTML output: %s", out)
	}
}

func TestCLI_MaxScoreThreshold(t *testing.T) {
	// replacement-plan has security group replace + compute updates (score >= 20)
	cmd := exec.Command(binaryPath, "--max-score", "5", filepath.Join("..", "..", "testdata", "replacement-plan.json"))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected exit code 1 when blast score exceeds max-score 5, but got exit 0")
	}

	out := stdout.String()
	if !strings.Contains(out, "Policy Check FAILED") {
		t.Errorf("expected policy failure in output: %s", out)
	}
}

func TestCLI_DiffCommand(t *testing.T) {
	plan1 := filepath.Join("..", "..", "testdata", "replacement-plan.json")
	plan2 := filepath.Join("..", "..", "testdata", "clean-plan.json")

	cmd := exec.Command(binaryPath, "diff", plan1, plan2)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("diff command failed: %v, stderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "Plan Comparison & Risk Delta") {
		t.Errorf("expected diff header in output: %s", out)
	}
	if !strings.Contains(out, "Mitigated / Resolved Risks") {
		t.Errorf("expected resolved risks in output: %s", out)
	}
}

func TestCLI_VersionFlag(t *testing.T) {
	cmd := exec.Command(binaryPath, "--version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to run with --version: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "tf-blast version") {
		t.Errorf("expected 'tf-blast version' in output, got: %s", out)
	}
}

func TestCLI_VersionCommand(t *testing.T) {
	cmd := exec.Command(binaryPath, "version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to run 'tf-blast version': %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "tf-blast version") {
		t.Errorf("expected 'tf-blast version' in output, got: %s", out)
	}
}
