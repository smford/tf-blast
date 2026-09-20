package renderer

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/smford/tf-blast/pkg/analyzer"
)

// SARIFReport represents OASIS SARIF v2.1.0 standard schema for GitHub Code Scanning.
type SARIFReport struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []SARIFRun `json:"runs"`
}

type SARIFRun struct {
	Tool    SARIFTool     `json:"tool"`
	Results []SARIFResult `json:"results"`
}

type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

type SARIFDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []SARIFRule `json:"rules"`
}

type SARIFRule struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	ShortDescription     SARIFMessage       `json:"shortDescription"`
	FullDescription      SARIFMessage       `json:"fullDescription"`
	DefaultConfiguration SARIFConfiguration `json:"defaultConfiguration"`
}

type SARIFConfiguration struct {
	Level string `json:"level"` // "error", "warning", "note"
}

type SARIFResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   SARIFMessage    `json:"message"`
	Locations []SARIFLocation `json:"locations,omitempty"`
}

type SARIFMessage struct {
	Text string `json:"text"`
}

type SARIFLocation struct {
	PhysicalLocation *SARIFPhysicalLocation `json:"physicalLocation,omitempty"`
	LogicalLocations []SARIFLogicalLocation `json:"logicalLocations,omitempty"`
}

type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
	Region           *SARIFRegion          `json:"region,omitempty"`
}

type SARIFArtifactLocation struct {
	URI string `json:"uri"`
}

type SARIFRegion struct {
	StartLine int `json:"startLine"`
}

type SARIFLogicalLocation struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// RenderSARIF converts the AnalysisReport to a valid SARIF 2.1.0 JSON document.
func RenderSARIF(w io.Writer, report *analyzer.AnalysisReport, toolVersion string) error {
	if toolVersion == "" {
		toolVersion = "1.0.0"
	}

	sarif := SARIFReport{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []SARIFRun{
			{
				Tool: SARIFTool{
					Driver: SARIFDriver{
						Name:           "tf-blast",
						Version:        toolVersion,
						InformationURI: "https://github.com/smford/tf-blast",
						Rules: []SARIFRule{
							{
								ID:                   "TF-BLAST-001",
								Name:                 "CriticalBlastRadius",
								ShortDescription:     SARIFMessage{Text: "Critical blast radius: Destructive deletion or replacement of stateful/storage resources."},
								FullDescription:      SARIFMessage{Text: "Destructive modifications to data stores, databases, or stateful volumes introduce high operational risk and potential data loss."},
								DefaultConfiguration: SARIFConfiguration{Level: "error"},
							},
							{
								ID:                   "TF-BLAST-002",
								Name:                 "HighBlastRadius",
								ShortDescription:     SARIFMessage{Text: "High blast radius: Severed dependencies or routing/identity disruptions."},
								FullDescription:      SARIFMessage{Text: "Modifications to security groups, route tables, or IAM policies that downstream compute resources rely on."},
								DefaultConfiguration: SARIFConfiguration{Level: "warning"},
							},
							{
								ID:                   "TF-BLAST-003",
								Name:                 "DestructiveReplacement",
								ShortDescription:     SARIFMessage{Text: "Destructive resource replacement (delete-then-create)."},
								FullDescription:      SARIFMessage{Text: "Terraform will drop and recreate this resource due to forced attribute replacement."},
								DefaultConfiguration: SARIFConfiguration{Level: "warning"},
							},
							{
								ID:                   "TF-BLAST-004",
								Name:                 "StateDriftDetected",
								ShortDescription:     SARIFMessage{Text: "Out-of-band console drift detected."},
								FullDescription:      SARIFMessage{Text: "This resource was modified outside of Terraform in the cloud console or drifted from state."},
								DefaultConfiguration: SARIFConfiguration{Level: "warning"},
							},
						},
					},
				},
				Results: make([]SARIFResult, 0),
			},
		},
	}

	if report != nil {
		for _, res := range report.Resources {
			// Skip low-risk creations or clean items from SARIF annotations
			if res.Action == analyzer.ActionCreate || res.Action == analyzer.ActionNoop {
				continue
			}

			var ruleID string
			var level string

			switch res.Severity {
			case analyzer.SeverityCritical:
				ruleID = "TF-BLAST-001"
				level = "error"
			case analyzer.SeverityHigh:
				ruleID = "TF-BLAST-002"
				level = "warning"
			default:
				if res.Action == analyzer.ActionReplace {
					ruleID = "TF-BLAST-003"
					level = "warning"
				} else if res.HasDrift {
					ruleID = "TF-BLAST-004"
					level = "note"
				} else {
					continue
				}
			}

			msgText := fmt.Sprintf("[%s] %s: %s planned", res.Severity, res.Address, res.Action)
			if res.RootCause != "" && res.RootCause != "-" {
				msgText += fmt.Sprintf(" (Root cause: %s)", res.RootCause)
			}
			if res.DownstreamCount > 0 {
				msgText += fmt.Sprintf(". Cascades to %d downstream dependent resource(s): %s",
					res.DownstreamCount, strings.Join(res.DownstreamAffected, ", "))
			}
			if res.HasDrift {
				msgText += fmt.Sprintf(". Note: %s", res.DriftDetails)
			}

			uri := "main.tf"
			if res.PlanSource != "" && res.PlanSource != "stdin" {
				uri = res.PlanSource + "/main.tf"
			}

			result := SARIFResult{
				RuleID:  ruleID,
				Level:   level,
				Message: SARIFMessage{Text: msgText},
				Locations: []SARIFLocation{
					{
						PhysicalLocation: &SARIFPhysicalLocation{
							ArtifactLocation: SARIFArtifactLocation{URI: uri},
							Region:           &SARIFRegion{StartLine: 1},
						},
						LogicalLocations: []SARIFLogicalLocation{
							{
								Name: res.Address,
								Kind: "resource",
							},
						},
					},
				},
			}

			sarif.Runs[0].Results = append(sarif.Runs[0].Results, result)
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(sarif)
}
