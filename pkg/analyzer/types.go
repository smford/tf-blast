package analyzer

import (
	"github.com/smford/tf-blast/pkg/graph"
	"github.com/smford/tf-blast/pkg/parser"
)

// ActionType classifies the planned change operation.
type ActionType string

const (
	ActionNoop          ActionType = "NOOP"
	ActionCreate        ActionType = "CREATE"
	ActionUpdateInPlace ActionType = "UPDATE_IN_PLACE"
	ActionReplace       ActionType = "REPLACE"
	ActionDestroy       ActionType = "DESTROY"
	ActionRead          ActionType = "READ"
)

// Severity represents the semantic risk level.
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityClean    Severity = "CLEAN"
)

// ResourceAnalysis encapsulates the evaluation of a single resource.
type ResourceAnalysis struct {
	Address            string                 `json:"address"`
	Type               string                 `json:"type"`
	Name               string                 `json:"name"`
	Module             string                 `json:"module,omitempty"`
	Action             ActionType             `json:"action"`
	Severity           Severity               `json:"severity"`
	RootCause          string                 `json:"root_cause,omitempty"`
	ReplacePaths       []string               `json:"replace_paths,omitempty"`
	DownstreamAffected []string               `json:"downstream_affected,omitempty"`
	DownstreamCount    int                    `json:"downstream_count"`
	DownstreamTree     *graph.TreeNode        `json:"downstream_tree,omitempty"`
	HasDrift           bool                   `json:"has_drift,omitempty"`
	DriftDetails       string                 `json:"drift_details,omitempty"`
	IsSensitive        bool                   `json:"is_sensitive,omitempty"`
	PlanSource         string                 `json:"plan_source,omitempty"`
	RawChange          *parser.ResourceChange `json:"-"`
}

// Summary provides aggregate metrics of the plan analysis.
type Summary struct {
	ToAdd            int      `json:"to_add"`
	ToUpdate         int      `json:"to_update"`
	ToDestroy        int      `json:"to_destroy"`
	ToReplace        int      `json:"to_replace"`
	Unchanged        int      `json:"unchanged"`
	TotalChanges     int      `json:"total_changes"`
	TotalBlastRadius int      `json:"total_blast_radius"`
	DriftCount       int      `json:"drift_count,omitempty"`
	MaxSeverity      Severity `json:"max_severity"`
	PlanHealth       string   `json:"plan_health"`
}

// AnalysisReport is the complete result returned by the analyzer.
type AnalysisReport struct {
	Summary    Summary            `json:"summary"`
	Resources  []ResourceAnalysis `json:"resources"`
	Failed     bool               `json:"failed"`
	FailReason string             `json:"fail_reason,omitempty"`
}
