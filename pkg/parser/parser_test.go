package parser

import (
	"strings"
	"testing"
)

func TestParsePlan_Simple(t *testing.T) {
	jsonPlan := `{
		"format_version": "1.2",
		"terraform_version": "1.7.0",
		"resource_changes": [
			{
				"address": "aws_security_group.db",
				"mode": "managed",
				"type": "aws_security_group",
				"name": "db",
				"provider_name": "registry.terraform.io/hashicorp/aws",
				"change": {
					"actions": ["delete", "create"],
					"before": {"id": "sg-123", "name": "db-sg"},
					"after": {"name": "db-sg-v2"},
					"replace_paths": [["name"]]
				}
			}
		],
		"configuration": {
			"root_module": {
				"resources": [
					{
						"address": "aws_instance.app",
						"mode": "managed",
						"type": "aws_instance",
						"name": "app",
						"expressions": {
							"vpc_security_group_ids": {
								"references": ["aws_security_group.db.id"]
							}
						}
					}
				]
			}
		}
	}`

	plan, err := ParsePlan(strings.NewReader(jsonPlan))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(plan.ResourceChanges) != 1 {
		t.Fatalf("expected 1 resource change, got %d", len(plan.ResourceChanges))
	}

	rc := plan.ResourceChanges[0]
	if rc.Address != "aws_security_group.db" {
		t.Errorf("expected address aws_security_group.db, got %s", rc.Address)
	}

	if len(rc.Change.ReplacePaths) != 1 {
		t.Fatalf("expected 1 replace_path, got %d", len(rc.Change.ReplacePaths))
	}

	formatted := FormatReplacePath(rc.Change.ReplacePaths[0])
	if formatted != "name" {
		t.Errorf("expected formatted path 'name', got '%s'", formatted)
	}

	deps := ExtractConfigDependencies(plan)
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}
	if deps[0].Source != "aws_instance.app" || deps[0].Target != "aws_security_group.db" {
		t.Errorf("unexpected dependency: %+v", deps[0])
	}
}

func TestFormatReplacePath_Complex(t *testing.T) {
	path := []any{"ingress", float64(0), "cidr_blocks"}
	res := FormatReplacePath(path)
	if res != "ingress[0].cidr_blocks" {
		t.Errorf("expected ingress[0].cidr_blocks, got %s", res)
	}
}

func TestExtractStateRelationships(t *testing.T) {
	jsonPlan := `{
		"format_version": "1.2",
		"resource_changes": [
			{
				"address": "aws_vpc.main",
				"type": "aws_vpc",
				"change": {
					"actions": ["no-op"],
					"before": {"id": "vpc-0123456789abcdef0"}
				}
			},
			{
				"address": "aws_subnet.sub1",
				"type": "aws_subnet",
				"change": {
					"actions": ["create"],
					"after": {"vpc_id": "vpc-0123456789abcdef0"}
				}
			}
		]
	}`

	plan, err := ParsePlan(strings.NewReader(jsonPlan))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	refs := ExtractStateRelationships(plan)
	if len(refs) != 1 {
		t.Fatalf("expected 1 extracted relationship, got %d", len(refs))
	}
	if refs[0].Source != "aws_subnet.sub1" || refs[0].Target != "aws_vpc.main" {
		t.Errorf("expected aws_subnet.sub1 -> aws_vpc.main, got %+v", refs[0])
	}
}
