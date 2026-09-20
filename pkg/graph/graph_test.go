package graph

import (
	"strings"
	"testing"

	"github.com/smford/tf-blast/pkg/parser"
)

func TestGraph_BasicEdgesAndDownstream(t *testing.T) {
	g := NewGraph()

	// A depends on B, B depends on C
	// Downstream: C changes -> B impacted -> A impacted
	g.AddEdge("aws_instance.app", "aws_security_group.db")
	g.AddEdge("aws_security_group.db", "aws_vpc.main")

	if g.NodeCount() != 3 {
		t.Fatalf("expected 3 nodes, got %d", g.NodeCount())
	}
	if g.EdgeCount() != 2 {
		t.Fatalf("expected 2 edges, got %d", g.EdgeCount())
	}

	dsC := g.GetTransitiveDownstream("aws_vpc.main")
	if len(dsC) != 2 {
		t.Fatalf("expected 2 transitive downstream nodes for aws_vpc.main, got %d", len(dsC))
	}
	if dsC[0] != "aws_instance.app" || dsC[1] != "aws_security_group.db" {
		t.Errorf("unexpected downstream: %v", dsC)
	}

	dsB := g.GetTransitiveDownstream("aws_security_group.db")
	if len(dsB) != 1 || dsB[0] != "aws_instance.app" {
		t.Fatalf("expected [aws_instance.app] for aws_security_group.db, got %v", dsB)
	}

	usA := g.GetTransitiveUpstream("aws_instance.app")
	if len(usA) != 2 {
		t.Fatalf("expected 2 upstream nodes for aws_instance.app, got %d", len(usA))
	}
}

func TestGraph_CycleDetection(t *testing.T) {
	g := NewGraph()
	// A -> B -> C -> A
	g.AddEdge("resA", "resB")
	g.AddEdge("resB", "resC")
	g.AddEdge("resC", "resA")

	if !g.HasCycles() {
		t.Errorf("expected graph to have cycles")
	}

	// BFS should still terminate safely without infinite loop
	ds := g.GetTransitiveDownstream("resA")
	if len(ds) != 2 {
		t.Errorf("expected 2 downstream nodes in cycle, got %d (%v)", len(ds), ds)
	}
}

func TestGraph_TreeRendering(t *testing.T) {
	g := NewGraph()
	g.AddEdge("module.api.aws_ecs_service.web", "aws_rds_cluster.primary")
	g.AddEdge("module.worker.aws_ecs_service.processor", "aws_rds_cluster.primary")

	tree := g.BuildDownstreamTree("aws_rds_cluster.primary", 5)
	rendered := tree.RenderTree()

	if !strings.Contains(rendered, "aws_rds_cluster.primary") {
		t.Errorf("missing root in rendered tree: %s", rendered)
	}
	if !strings.Contains(rendered, "module.api.aws_ecs_service.web") {
		t.Errorf("missing child 1 in rendered tree: %s", rendered)
	}
	if !strings.Contains(rendered, "module.worker.aws_ecs_service.processor") {
		t.Errorf("missing child 2 in rendered tree: %s", rendered)
	}
	if !strings.Contains(rendered, "├── ") || !strings.Contains(rendered, "└── ") {
		t.Errorf("missing tree branch markers in rendered tree:\n%s", rendered)
	}
}

func TestBuildGraph_FromPlan(t *testing.T) {
	plan := &parser.Plan{
		FormatVersion: "1.2",
		ResourceChanges: []parser.ResourceChange{
			{
				Address: "aws_security_group.db",
				Type:    "aws_security_group",
				Name:    "db",
				Change: parser.Change{
					Actions: []string{"delete", "create"},
				},
			},
		},
		Configuration: &parser.Configuration{
			RootModule: parser.ConfigModule{
				Resources: []parser.ConfigResource{
					{
						Address: "aws_instance.app",
						Type:    "aws_instance",
						Name:    "app",
						Expressions: map[string]parser.Expression{
							"vpc_security_group_ids": {
								References: []string{"aws_security_group.db.id"},
							},
						},
					},
				},
			},
		},
	}

	g := BuildGraph(plan)
	if g.NodeCount() != 2 {
		t.Fatalf("expected 2 nodes, got %d", g.NodeCount())
	}

	ds := g.GetDirectDownstream("aws_security_group.db")
	if len(ds) != 1 || ds[0] != "aws_instance.app" {
		t.Fatalf("expected aws_instance.app downstream of aws_security_group.db, got %v", ds)
	}
}
