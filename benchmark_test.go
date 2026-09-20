package main_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/smford/tf-blast/pkg/analyzer"
	"github.com/smford/tf-blast/pkg/config"
	"github.com/smford/tf-blast/pkg/graph"
	"github.com/smford/tf-blast/pkg/parser"
	"github.com/smford/tf-blast/pkg/renderer"
)

func TestMassivePlan_Sub500msExecution(t *testing.T) {
	planPath := filepath.Join("testdata", "massive-plan.json")
	file, err := os.Open(planPath)
	if err != nil {
		t.Fatalf("failed to open massive-plan.json: %v", err)
	}
	defer file.Close()

	start := time.Now()

	// 1. Ingestion
	plan, err := parser.ParsePlan(file)
	if err != nil {
		t.Fatalf("failed to parse plan: %v", err)
	}

	// 2. Graph construction
	g := graph.BuildGraph(plan)

	// 3. Risk & blast radius analysis
	cfg := config.DefaultConfig()
	report := analyzer.Analyze(plan, g, cfg)

	// 4. Render to markdown
	_ = renderer.RenderMarkdown(io.Discard, report)

	duration := time.Since(start)

	if len(plan.ResourceChanges) < 1000 {
		t.Fatalf("expected at least 1000 resources, got %d", len(plan.ResourceChanges))
	}

	t.Logf("Processed %d resources in %v (requirement: < 500ms)", len(plan.ResourceChanges), duration)

	if duration > 500*time.Millisecond {
		t.Errorf("execution took %v, exceeding sub-500ms limit", duration)
	}
}

func BenchmarkFullPipeline_MassivePlan(b *testing.B) {
	planPath := filepath.Join("testdata", "massive-plan.json")
	cfg := config.DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file, err := os.Open(planPath)
		if err != nil {
			b.Fatalf("failed to open plan: %v", err)
		}

		plan, err := parser.ParsePlan(file)
		if err != nil {
			file.Close()
			b.Fatalf("failed to parse plan: %v", err)
		}
		file.Close()

		g := graph.BuildGraph(plan)
		_ = analyzer.Analyze(plan, g, cfg)
	}
}
