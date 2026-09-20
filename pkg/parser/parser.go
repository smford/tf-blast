package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ParsePlan reads and decodes a Terraform or OpenTofu execution plan from an io.Reader.
func ParsePlan(r io.Reader) (*Plan, error) {
	var plan Plan
	dec := json.NewDecoder(r)
	if err := dec.Decode(&plan); err != nil {
		return nil, fmt.Errorf("failed to decode terraform json plan: %w", err)
	}

	if plan.FormatVersion == "" && len(plan.ResourceChanges) == 0 && plan.Configuration == nil {
		return nil, fmt.Errorf("invalid plan: missing format_version, resource_changes, and configuration")
	}

	return &plan, nil
}

// ExtractedReference holds a dependency from a source resource to a target resource.
type ExtractedReference struct {
	Source      string // The resource that depends on Target
	Target      string // The resource being depended upon
	SourceField string
	IsExplicit  bool
}

// ExtractConfigDependencies traverses the plan's configuration tree and extracts
// all explicit (depends_on) and implicit (expressions.*.references) resource-level dependencies.
func ExtractConfigDependencies(plan *Plan) []ExtractedReference {
	if plan == nil || plan.Configuration == nil {
		return nil
	}

	var refs []ExtractedReference
	extractModuleConfig(&plan.Configuration.RootModule, "", &refs)
	return refs
}

func extractModuleConfig(mod *ConfigModule, modulePrefix string, refs *[]ExtractedReference) {
	if mod == nil {
		return
	}

	// 1. Process resources in this module
	for _, res := range mod.Resources {
		srcAddr := qualifyAddress(modulePrefix, res.Address)

		// Explicit depends_on
		for _, dep := range res.DependsOn {
			targetAddr := qualifyAddress(modulePrefix, dep)
			*refs = append(*refs, ExtractedReference{
				Source:     srcAddr,
				Target:     targetAddr,
				IsExplicit: true,
			})
		}

		// Implicit expression references
		for exprName, expr := range res.Expressions {
			for _, ref := range expr.References {
				target := resolveReference(modulePrefix, ref)
				if target != "" && target != srcAddr {
					*refs = append(*refs, ExtractedReference{
						Source:      srcAddr,
						Target:      target,
						SourceField: exprName,
						IsExplicit:  false,
					})
				}
			}
		}
	}

	// 2. Process child module calls
	for callName, call := range mod.ModuleCalls {
		childPrefix := callName
		if modulePrefix != "" {
			childPrefix = modulePrefix + ".module." + callName
		} else {
			childPrefix = "module." + callName
		}

		// Check module call expressions (passing inputs to module)
		for _, expr := range call.Expressions {
			for _, ref := range expr.References {
				target := resolveReference(modulePrefix, ref)
				if target != "" {
					*refs = append(*refs, ExtractedReference{
						Source:     childPrefix,
						Target:     target,
						IsExplicit: false,
					})
				}
			}
		}

		// Explicit module depends_on
		for _, dep := range call.DependsOn {
			targetAddr := qualifyAddress(modulePrefix, dep)
			*refs = append(*refs, ExtractedReference{
				Source:     childPrefix,
				Target:     targetAddr,
				IsExplicit: true,
			})
		}

		if call.Module != nil {
			extractModuleConfig(call.Module, childPrefix, refs)
		}
	}
}

// qualifyAddress adds a module prefix to a resource address if not already prefixed.
func qualifyAddress(modulePrefix, address string) string {
	cleanAddr := strings.TrimSpace(address)
	if cleanAddr == "" {
		return ""
	}
	if strings.HasPrefix(cleanAddr, "module.") || modulePrefix == "" {
		return cleanAddr
	}
	return modulePrefix + "." + cleanAddr
}

// resolveReference converts an expression reference (e.g. "aws_security_group.db.id" or "module.vpc.subnet_id")
// into a target resource or module address.
func resolveReference(modulePrefix, rawRef string) string {
	ref := strings.TrimSpace(rawRef)
	if ref == "" {
		return ""
	}

	// Skip variables, locals, paths, each, count
	if strings.HasPrefix(ref, "var.") ||
		strings.HasPrefix(ref, "local.") ||
		strings.HasPrefix(ref, "path.") ||
		strings.HasPrefix(ref, "each.") ||
		strings.HasPrefix(ref, "count.") {
		return ""
	}

	// If it starts with module.
	if strings.HasPrefix(ref, "module.") {
		parts := strings.Split(ref, ".")
		if len(parts) >= 2 {
			modAddr := parts[0] + "." + parts[1]
			if modulePrefix != "" {
				return modulePrefix + "." + modAddr
			}
			return modAddr
		}
	}

	// If data source reference: data.aws_ami.ubuntu.id -> data.aws_ami.ubuntu
	if strings.HasPrefix(ref, "data.") {
		parts := strings.Split(ref, ".")
		if len(parts) >= 3 {
			dataAddr := parts[0] + "." + parts[1] + "." + parts[2]
			return qualifyAddress(modulePrefix, dataAddr)
		}
	}

	// Standard resource: aws_security_group.db.id -> aws_security_group.db
	parts := strings.Split(ref, ".")
	if len(parts) >= 2 {
		resAddr := parts[0] + "." + parts[1]
		// Strip indexing like [0] from resource part if present
		return qualifyAddress(modulePrefix, resAddr)
	}

	return ""
}

// ExtractStateRelationships inspects prior_state and planned_values to correlate
// resources by shared ID / ARN values (e.g., resource B's VPC ID matches resource A's ID).
func ExtractStateRelationships(plan *Plan) []ExtractedReference {
	if plan == nil {
		return nil
	}

	// Map of unique identifier -> resource address
	identifiers := make(map[string]string)

	// Harvest IDs from prior state
	if plan.PriorState != nil {
		harvestModuleIdentifiers(getPriorStateRoot(plan.PriorState), "", identifiers)
	}
	// Harvest IDs from planned values
	if plan.PlannedValues != nil {
		harvestModuleIdentifiers(&plan.PlannedValues.RootModule, "", identifiers)
	}

	// Harvest IDs from resource changes (before and after values)
	for _, rc := range plan.ResourceChanges {
		addr := StripIndex(rc.Address)
		harvestMapIdentifiers(rc.Change.Before, addr, identifiers)
		harvestMapIdentifiers(rc.Change.After, addr, identifiers)
	}

	// Now scan resource values for occurrences of known identifiers
	var refs []ExtractedReference
	seen := make(map[string]bool)

	checkMatch := func(srcAddr, val string) {
		val = strings.TrimSpace(val)
		if val == "" || len(val) < 4 {
			return
		}
		if targetAddr, found := identifiers[val]; found {
			if targetAddr != srcAddr && !strings.HasPrefix(srcAddr, targetAddr) {
				key := srcAddr + "->" + targetAddr
				if !seen[key] {
					seen[key] = true
					refs = append(refs, ExtractedReference{
						Source:     srcAddr,
						Target:     targetAddr,
						IsExplicit: false,
					})
				}
			}
		}
	}

	// Search in resource changes
	for _, rc := range plan.ResourceChanges {
		addr := StripIndex(rc.Address)
		scanForIdentifiers(rc.Change.Before, addr, checkMatch)
		scanForIdentifiers(rc.Change.After, addr, checkMatch)
	}

	return refs
}

func getPriorStateRoot(ps *PriorState) *StateModule {
	if ps == nil {
		return nil
	}
	if ps.Values != nil {
		return &ps.Values.RootModule
	}
	return ps.RootModule
}

func harvestModuleIdentifiers(mod *StateModule, prefix string, identifiers map[string]string) {
	if mod == nil {
		return
	}
	for _, res := range mod.Resources {
		addr := StripIndex(res.Address)
		harvestMapIdentifiers(res.Values, addr, identifiers)
	}
	for _, child := range mod.ChildModules {
		harvestModuleIdentifiers(&child, prefix, identifiers)
	}
}

func harvestMapIdentifiers(values map[string]any, addr string, identifiers map[string]string) {
	if len(values) == 0 || addr == "" {
		return
	}
	// A resource only defines its own unique identity: typically "id", "arn", or "name"
	idKeys := []string{"id", "arn"}
	for _, key := range idKeys {
		if val, ok := values[key]; ok {
			if strVal, isStr := val.(string); isStr && len(strVal) > 3 {
				if !isGenericValue(strVal) {
					identifiers[strVal] = addr
				}
			}
		}
	}
}

func scanForIdentifiers(v any, srcAddr string, matchFn func(string, string)) {
	if v == nil {
		return
	}
	switch val := v.(type) {
	case string:
		matchFn(srcAddr, val)
	case []any:
		for _, item := range val {
			scanForIdentifiers(item, srcAddr, matchFn)
		}
	case map[string]any:
		for _, item := range val {
			scanForIdentifiers(item, srcAddr, matchFn)
		}
	}
}

func isGenericValue(s string) bool {
	lower := strings.ToLower(s)
	switch lower {
	case "true", "false", "null", "default", "none", "0", "1", "-1", "aws", "standard":
		return true
	default:
		return false
	}
}

// StripIndex removes array/map index notation from the end of a resource address (e.g. `aws_subnet.private[0]` -> `aws_subnet.private`).
func StripIndex(addr string) string {
	idx := strings.LastIndex(addr, "[")
	if idx != -1 && strings.HasSuffix(addr, "]") {
		bracketContent := addr[idx+1 : len(addr)-1]
		if !strings.Contains(bracketContent, " ") {
			return addr[:idx]
		}
	}
	return addr
}
