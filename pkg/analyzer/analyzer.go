package analyzer

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/smford/tf-blast/pkg/config"
	"github.com/smford/tf-blast/pkg/graph"
	"github.com/smford/tf-blast/pkg/parser"
)

// Analyze performs semantic risk analysis on a parsed Terraform plan and its dependency graph.
func Analyze(plan *parser.Plan, g *graph.Graph, cfg *config.Config) *AnalysisReport {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if g == nil {
		g = graph.BuildGraph(plan)
	}

	report := &AnalysisReport{
		Summary: Summary{
			MaxSeverity: SeverityClean,
		},
		Resources: make([]ResourceAnalysis, 0),
	}

	if plan == nil || len(plan.ResourceChanges) == 0 {
		report.Summary.PlanHealth = "CLEAN / NO CHANGES"
		return report
	}

	allImpactedNodes := make(map[string]bool)

	// Build map of out-of-band state drifts
	driftMap := make(map[string]*parser.ResourceDrift)
	if plan != nil {
		for i := range plan.ResourceDrift {
			d := &plan.ResourceDrift[i]
			driftMap[parser.StripIndex(d.Address)] = d
		}
	}

	for i := range plan.ResourceChanges {
		rc := &plan.ResourceChanges[i]
		cleanAddr := parser.StripIndex(rc.Address)

		// Check ignored resources
		if isIgnored(cleanAddr, rc.Type, cfg.IgnoreResources) {
			continue
		}

		action := classifyAction(rc.Change.Actions)
		rootCause, replacePaths, isSensitive := determineRootCause(rc, action)

		var downstream []string
		var downstreamTree *graph.TreeNode
		if action == ActionReplace || action == ActionDestroy || (action == ActionUpdateInPlace && isNetworkOrIdentity(rc.Type, cfg)) {
			downstream = g.GetTransitiveDownstream(cleanAddr)
			if len(downstream) > 0 {
				downstreamTree = g.BuildDownstreamTree(cleanAddr, 4)
			}
		}

		// Calculate blast radius contribution
		if action == ActionReplace || action == ActionDestroy {
			allImpactedNodes[cleanAddr] = true
			for _, d := range downstream {
				allImpactedNodes[d] = true
			}
		}

		severity := evaluateSeverity(rc, action, len(downstream), downstream, g, cfg)

		// Check for detected drift
		hasDrift := false
		driftDetails := ""
		if d, exists := driftMap[cleanAddr]; exists {
			hasDrift = true
			report.Summary.DriftCount++
			driftDetails = "Out-of-band drift detected"
			if len(d.Change.Actions) > 0 {
				driftDetails += fmt.Sprintf(" (%s)", strings.Join(d.Change.Actions, ", "))
			}
		}

		// Update metrics
		switch action {
		case ActionCreate:
			report.Summary.ToAdd++
		case ActionUpdateInPlace:
			report.Summary.ToUpdate++
		case ActionDestroy:
			report.Summary.ToDestroy++
		case ActionReplace:
			report.Summary.ToReplace++
		case ActionNoop:
			report.Summary.Unchanged++
		}

		if action != ActionNoop && action != ActionRead {
			report.Summary.TotalChanges++
		}

		if severityRank(severity) > severityRank(report.Summary.MaxSeverity) {
			report.Summary.MaxSeverity = severity
		}

		riskScore := calculateRiskScore(severity, action, len(downstream), hasDrift)
		report.Summary.BlastScore += riskScore

		resAnalysis := ResourceAnalysis{
			Address:            cleanAddr,
			Type:               rc.Type,
			Name:               rc.Name,
			Module:             rc.ModuleAddress,
			Action:             action,
			Severity:           severity,
			RootCause:          rootCause,
			ReplacePaths:       replacePaths,
			DownstreamAffected: downstream,
			DownstreamCount:    len(downstream),
			DownstreamTree:     downstreamTree,
			HasDrift:           hasDrift,
			DriftDetails:       driftDetails,
			IsSensitive:        isSensitive,
			PlanSource:         rc.PlanSource,
			RiskScore:          riskScore,
			RawChange:          rc,
		}

		report.Resources = append(report.Resources, resAnalysis)
	}

	// Sort resources: highest severity first, then highest risk score, then address
	sort.Slice(report.Resources, func(i, j int) bool {
		r1, r2 := report.Resources[i], report.Resources[j]
		if severityRank(r1.Severity) != severityRank(r2.Severity) {
			return severityRank(r1.Severity) > severityRank(r2.Severity)
		}
		if r1.RiskScore != r2.RiskScore {
			return r1.RiskScore > r2.RiskScore
		}
		if r1.DownstreamCount != r2.DownstreamCount {
			return r1.DownstreamCount > r2.DownstreamCount
		}
		return r1.Address < r2.Address
	})

	report.Summary.TotalBlastRadius = len(allImpactedNodes)
	report.Summary.PlanHealth = determinePlanHealth(report.Summary.MaxSeverity, report.Summary.TotalBlastRadius, report.Summary.BlastScore, report.Summary.TotalChanges)

	// Threshold evaluation
	evaluateThresholds(report, cfg)

	return report
}

func classifyAction(actions []string) ActionType {
	if len(actions) == 0 {
		return ActionNoop
	}
	if len(actions) == 1 {
		switch actions[0] {
		case "no-op":
			return ActionNoop
		case "create":
			return ActionCreate
		case "update":
			return ActionUpdateInPlace
		case "delete":
			return ActionDestroy
		case "read":
			return ActionRead
		}
	}
	if len(actions) == 2 {
		if (actions[0] == "delete" && actions[1] == "create") ||
			(actions[0] == "create" && actions[1] == "delete") {
			return ActionReplace
		}
	}
	return ActionUpdateInPlace
}

func determineRootCause(rc *parser.ResourceChange, action ActionType) (string, []string, bool) {
	isSensitive := false

	switch action {
	case ActionCreate:
		return "Net-new resource", nil, false
	case ActionDestroy:
		if rc.ActionReason != "" {
			return fmt.Sprintf("Deletion (%s)", rc.ActionReason), nil, false
		}
		return "Dropped in PR", nil, false
	case ActionReplace:
		var formattedPaths []string
		for _, p := range rc.Change.ReplacePaths {
			pathStr := parser.FormatReplacePath(p)
			if parser.IsFieldSensitive(pathStr, rc.Change.BeforeSensitive) || parser.IsFieldSensitive(pathStr, rc.Change.AfterSensitive) {
				isSensitive = true
				pathStr += " (sensitive)"
			}
			formattedPaths = append(formattedPaths, pathStr)
		}

		// Check for specific version downgrades or obvious attribute diffs
		if len(formattedPaths) > 0 {
			firstAttr := formattedPaths[0]
			if firstAttr == "engine_version" && rc.Change.Before != nil && rc.Change.After != nil {
				beforeVer := fmt.Sprintf("%v", rc.Change.Before["engine_version"])
				afterVer := fmt.Sprintf("%v", rc.Change.After["engine_version"])
				if afterVer < beforeVer {
					return "engine_version (downgrade)", formattedPaths, isSensitive
				}
			}
			return strings.Join(formattedPaths, ", "), formattedPaths, isSensitive
		}

		if rc.ActionReason != "" {
			if strings.Contains(rc.ActionReason, "tainted") {
				return "Resource tainted", nil, false
			}
			return rc.ActionReason, nil, false
		}
		if rc.Change.ActionReason != "" {
			return rc.Change.ActionReason, nil, false
		}
		return "Forces new resource", nil, false

	case ActionUpdateInPlace:
		// Extract modified attributes with sensitive value masking
		var modified []string
		if rc.Change.Before != nil && rc.Change.After != nil {
			for k, vAfter := range rc.Change.After {
				vBefore, exists := rc.Change.Before[k]
				if !exists || fmt.Sprintf("%v", vBefore) != fmt.Sprintf("%v", vAfter) {
					if parser.IsFieldSensitive(k, rc.Change.BeforeSensitive) || parser.IsFieldSensitive(k, rc.Change.AfterSensitive) {
						isSensitive = true
						modified = append(modified, fmt.Sprintf("%s (sensitive redacted)", k))
					} else {
						modified = append(modified, k)
					}
				}
			}
			sort.Strings(modified)
		}
		if len(modified) > 0 {
			if len(modified) > 3 {
				return fmt.Sprintf("%s and %d others", strings.Join(modified[:3], ", "), len(modified)-3), nil, isSensitive
			}
			return strings.Join(modified, ", "), nil, isSensitive
		}
		return "In-place modification", nil, isSensitive

	default:
		return "-", nil, false
	}
}

func evaluateSeverity(
	rc *parser.ResourceChange,
	action ActionType,
	downstreamCount int,
	downstream []string,
	g *graph.Graph,
	cfg *config.Config,
) Severity {
	if action == ActionNoop || action == ActionRead {
		return SeverityClean
	}
	if action == ActionCreate {
		return SeverityLow
	}

	resType := strings.ToLower(rc.Type)

	// Check for disruptive in-place modifications (instance resizing, DB parameter group changes)
	if action == ActionUpdateInPlace && rc.Change.Before != nil && rc.Change.After != nil {
		if bType, ok1 := rc.Change.Before["instance_type"]; ok1 {
			if aType, ok2 := rc.Change.After["instance_type"]; ok2 && fmt.Sprintf("%v", bType) != fmt.Sprintf("%v", aType) {
				if downstreamCount >= 3 {
					return SeverityHigh
				}
				return SeverityMedium
			}
		}
		if bClass, ok1 := rc.Change.Before["instance_class"]; ok1 {
			if aClass, ok2 := rc.Change.After["instance_class"]; ok2 && fmt.Sprintf("%v", bClass) != fmt.Sprintf("%v", aClass) {
				return SeverityMedium
			}
		}
		if bParam, ok1 := rc.Change.Before["parameter_group_name"]; ok1 {
			if aParam, ok2 := rc.Change.After["parameter_group_name"]; ok2 && fmt.Sprintf("%v", bParam) != fmt.Sprintf("%v", aParam) {
				return SeverityHigh
			}
		}
	}

	// Check Critical Rules
	if matchAny(resType, cfg.Rules.Critical) {
		if action == ActionReplace || action == ActionDestroy {
			return SeverityCritical
		}
		return SeverityMedium
	}

	// Check High Rules
	if matchAny(resType, cfg.Rules.High) {
		if action == ActionReplace || action == ActionDestroy {
			// If it cascades to downstream stateful or compute resources, keep High or Critical
			if downstreamCount >= 5 {
				return SeverityCritical
			}
			return SeverityHigh
		}
		// In-place update to security group / route table with downstream dependents
		if downstreamCount > 0 {
			return SeverityHigh
		}
		return SeverityMedium
	}

	// Check Medium Rules
	if matchAny(resType, cfg.Rules.Medium) {
		if action == ActionReplace || action == ActionDestroy {
			if downstreamCount >= 5 {
				return SeverityHigh
			}
			return SeverityMedium
		}
		return SeverityLow
	}

	// Check Low Rules
	if matchAny(resType, cfg.Rules.Low) {
		return SeverityLow
	}

	// Default fallback based on action
	switch action {
	case ActionDestroy, ActionReplace:
		if downstreamCount >= 5 {
			return SeverityHigh
		}
		return SeverityMedium
	case ActionUpdateInPlace:
		if downstreamCount >= 5 {
			return SeverityMedium
		}
		return SeverityLow
	default:
		return SeverityLow
	}
}

func isNetworkOrIdentity(resType string, cfg *config.Config) bool {
	resType = strings.ToLower(resType)
	return matchAny(resType, cfg.Rules.High)
}

func matchAny(target string, patterns []string) bool {
	target = strings.ToLower(target)
	for _, pattern := range patterns {
		pattern = strings.ToLower(pattern)
		// Try standard glob match
		if matched, _ := filepath.Match(pattern, target); matched {
			return true
		}
		// Substring fallback for convenience
		cleanPat := strings.Trim(pattern, "*")
		if cleanPat != "" && strings.Contains(target, cleanPat) {
			return true
		}
	}
	return false
}

func isIgnored(address, resType string, ignoredPatterns []string) bool {
	return matchAny(address, ignoredPatterns) || matchAny(resType, ignoredPatterns)
}

func severityRank(s Severity) int {
	switch s {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}

func calculateRiskScore(severity Severity, action ActionType, downstreamCount int, hasDrift bool) int {
	if action == ActionNoop || action == ActionRead {
		return 0
	}
	score := 0
	switch severity {
	case SeverityCritical:
		score += 25
	case SeverityHigh:
		score += 15
	case SeverityMedium:
		score += 5
	case SeverityLow:
		score += 1
	}

	if action == ActionReplace || action == ActionDestroy {
		score += 5
	}
	score += downstreamCount * 2
	if hasDrift {
		score += 3
	}
	return score
}

func determinePlanHealth(maxSev Severity, blastRadius int, blastScore int, totalChanges int) string {
	if totalChanges == 0 {
		return "CLEAN / NO CHANGES"
	}
	if maxSev == SeverityCritical || blastRadius >= 20 || blastScore >= 100 {
		return "CRITICAL BLAST RADIUS DETECTED"
	}
	if maxSev == SeverityHigh || blastRadius >= 10 || blastScore >= 50 {
		return "HIGH BLAST RADIUS DETECTED"
	}
	if maxSev == SeverityMedium || blastRadius > 0 || blastScore >= 20 {
		return "MODERATE BLAST RADIUS DETECTED"
	}
	return "LOW RISK / HEALTHY"
}

func evaluateThresholds(report *AnalysisReport, cfg *config.Config) {
	if cfg.MaxBlast > 0 && report.Summary.TotalBlastRadius > cfg.MaxBlast {
		report.Failed = true
		report.FailReason = fmt.Sprintf("Blast radius (%d) exceeds maximum acceptable limit (%d)", report.Summary.TotalBlastRadius, cfg.MaxBlast)
		return
	}

	if cfg.MaxScore > 0 && report.Summary.BlastScore > cfg.MaxScore {
		report.Failed = true
		report.FailReason = fmt.Sprintf("Blast score (%d) exceeds maximum acceptable limit (%d)", report.Summary.BlastScore, cfg.MaxScore)
		return
	}

	failOn := strings.ToLower(strings.TrimSpace(cfg.FailOn))
	if failOn == "" {
		return
	}

	switch failOn {
	case "critical":
		if report.Summary.MaxSeverity == SeverityCritical {
			report.Failed = true
			report.FailReason = "Critical severity changes detected"
		}
	case "high":
		if severityRank(report.Summary.MaxSeverity) >= severityRank(SeverityHigh) {
			report.Failed = true
			report.FailReason = "High severity or critical changes detected"
		}
	case "medium":
		if severityRank(report.Summary.MaxSeverity) >= severityRank(SeverityMedium) {
			report.Failed = true
			report.FailReason = "Medium or higher severity changes detected"
		}
	case "replacement":
		if report.Summary.ToReplace > 0 {
			report.Failed = true
			report.FailReason = fmt.Sprintf("%d destructive replacement(s) detected", report.Summary.ToReplace)
		}
	case "any-destroy":
		if report.Summary.ToDestroy > 0 || report.Summary.ToReplace > 0 {
			report.Failed = true
			report.FailReason = fmt.Sprintf("Destructive actions detected: %d destroy, %d replace", report.Summary.ToDestroy, report.Summary.ToReplace)
		}
	}
}
