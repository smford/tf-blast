package parser

import (
	"strings"
	"testing"
)

func TestPlanDisplayName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"environments/prod/vpc/plan.json", "environments/prod/vpc"},
		{"vpc/tfplan.json", "vpc"},
		{"testdata/clean-plan.json", "testdata/clean-plan"},
		{"plan.json", "plan"},
		{"-", "stdin"},
		{"", "stdin"},
		{"prod-plan.json", "prod-plan"},
	}

	for _, tc := range tests {
		actual := PlanDisplayName(tc.input)
		if actual != tc.expected {
			t.Errorf("PlanDisplayName(%q) = %q, expected %q", tc.input, actual, tc.expected)
		}
	}
}

func TestAggregatePlans_SinglePlan(t *testing.T) {
	planJSON := `{
		"format_version": "1.2",
		"resource_changes": [
			{
				"address": "aws_s3_bucket.data",
				"type": "aws_s3_bucket",
				"change": {"actions": ["create"]}
			}
		]
	}`

	p, err := ParsePlan(strings.NewReader(planJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	aggregated := AggregatePlans([]NamedPlan{
		{Name: "storage", Plan: p},
	})

	if len(aggregated.ResourceChanges) != 1 {
		t.Fatalf("expected 1 change, got %d", len(aggregated.ResourceChanges))
	}

	// In single plan, address should NOT be prefixed
	if aggregated.ResourceChanges[0].Address != "aws_s3_bucket.data" {
		t.Errorf("expected original address in single plan, got %s", aggregated.ResourceChanges[0].Address)
	}
	if aggregated.ResourceChanges[0].PlanSource != "storage" {
		t.Errorf("expected PlanSource 'storage', got %s", aggregated.ResourceChanges[0].PlanSource)
	}
}

func TestAggregatePlans_MultiPlan(t *testing.T) {
	plan1JSON := `{
		"format_version": "1.2",
		"resource_changes": [
			{
				"address": "aws_vpc.main",
				"type": "aws_vpc",
				"change": {
					"actions": ["create"],
					"after": {"id": "vpc-12345"}
				}
			}
		],
		"configuration": {
			"root_module": {
				"resources": [
					{
						"address": "aws_vpc.main",
						"type": "aws_vpc"
					}
				]
			}
		}
	}`

	plan2JSON := `{
		"format_version": "1.2",
		"resource_changes": [
			{
				"address": "aws_subnet.public",
				"type": "aws_subnet",
				"change": {
					"actions": ["create"],
					"after": {"vpc_id": "vpc-12345"}
				}
			}
		],
		"configuration": {
			"root_module": {
				"resources": [
					{
						"address": "aws_subnet.public",
						"type": "aws_subnet"
					}
				]
			}
		}
	}`

	p1, err := ParsePlan(strings.NewReader(plan1JSON))
	if err != nil {
		t.Fatalf("failed to parse plan 1: %v", err)
	}

	p2, err := ParsePlan(strings.NewReader(plan2JSON))
	if err != nil {
		t.Fatalf("failed to parse plan 2: %v", err)
	}

	aggregated := AggregatePlans([]NamedPlan{
		{Name: "network", Plan: p1},
		{Name: "compute", Plan: p2},
	})

	if len(aggregated.ResourceChanges) != 2 {
		t.Fatalf("expected 2 aggregated resource changes, got %d", len(aggregated.ResourceChanges))
	}

	rc1 := aggregated.ResourceChanges[0]
	if rc1.Address != "[network] aws_vpc.main" {
		t.Errorf("expected [network] aws_vpc.main, got %s", rc1.Address)
	}
	if rc1.PlanSource != "network" {
		t.Errorf("expected PlanSource 'network', got %s", rc1.PlanSource)
	}

	rc2 := aggregated.ResourceChanges[1]
	if rc2.Address != "[compute] aws_subnet.public" {
		t.Errorf("expected [compute] aws_subnet.public, got %s", rc2.Address)
	}
	if rc2.PlanSource != "compute" {
		t.Errorf("expected PlanSource 'compute', got %s", rc2.PlanSource)
	}

	// Verify cross-plan state relationship extraction
	refs := ExtractStateRelationships(aggregated)
	if len(refs) != 1 {
		t.Fatalf("expected 1 cross-plan state relationship, got %d", len(refs))
	}
	if refs[0].Source != "[compute] aws_subnet.public" || refs[0].Target != "[network] aws_vpc.main" {
		t.Errorf("expected [compute] aws_subnet.public -> [network] aws_vpc.main, got %+v", refs[0])
	}
}
