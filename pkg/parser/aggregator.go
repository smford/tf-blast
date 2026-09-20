package parser

import (
	"fmt"
	"path/filepath"
	"strings"
)

// NamedPlan pairs a parsed Plan with its identifier or file path.
type NamedPlan struct {
	Name string
	Plan *Plan
}

// PlanDisplayName derives a clean, human-readable stack or plan name from a file path.
// Examples:
//
//	"environments/prod/vpc/plan.json" -> "environments/prod/vpc"
//	"testdata/clean-plan.json"        -> "testdata/clean-plan"
//	"plan1.json"                      -> "plan1"
//	"-"                               -> "stdin"
func PlanDisplayName(path string) string {
	if path == "-" || path == "" {
		return "stdin"
	}
	clean := filepath.Clean(path)
	base := filepath.Base(clean)
	dir := filepath.Dir(clean)

	if (base == "plan.json" || base == "tfplan.json" || base == "plan.binary.json") && dir != "." {
		return dir
	}

	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if dir != "." && dir != "" {
		return dir + "/" + name
	}
	return name
}

// AggregatePlans merges multiple independent Terraform/OpenTofu plans into a single unified Plan.
// When multiple plans are provided, resources are qualified with `[stack-name]` tags, preserving
// intra-plan dependencies while enabling cross-plan relationship discovery via shared state identifiers (IDs/ARNs).
func AggregatePlans(namedPlans []NamedPlan) *Plan {
	var validPlans []NamedPlan
	for _, np := range namedPlans {
		if np.Plan != nil {
			validPlans = append(validPlans, np)
		}
	}

	if len(validPlans) == 0 {
		return &Plan{
			FormatVersion: "1.2",
		}
	}

	// Single plan: preserve original addresses and annotate plan_source
	if len(validPlans) == 1 {
		p := validPlans[0].Plan
		for i := range p.ResourceChanges {
			if p.ResourceChanges[i].PlanSource == "" {
				p.ResourceChanges[i].PlanSource = validPlans[0].Name
			}
		}
		return p
	}

	// Multiple plans: aggregate into consolidated structure
	formatVersion := "1.2"
	terraformVersion := ""
	for _, np := range validPlans {
		if np.Plan.FormatVersion != "" {
			formatVersion = np.Plan.FormatVersion
		}
		if np.Plan.TerraformVersion != "" {
			terraformVersion = np.Plan.TerraformVersion
		}
	}

	aggregated := &Plan{
		FormatVersion:    formatVersion,
		TerraformVersion: terraformVersion,
		Configuration: &Configuration{
			RootModule: ConfigModule{
				Resources:   make([]ConfigResource, 0),
				ModuleCalls: make(map[string]ModuleCall),
			},
		},
		PlannedValues: &PlannedValues{
			RootModule: StateModule{
				Resources:    make([]StateResource, 0),
				ChildModules: make([]StateModule, 0),
			},
		},
		PriorState: &PriorState{
			RootModule: &StateModule{
				Resources:    make([]StateResource, 0),
				ChildModules: make([]StateModule, 0),
			},
		},
	}

	for _, np := range validPlans {
		prefix := fmt.Sprintf("[%s] ", np.Name)

		// 1. Resource Changes
		for _, rc := range np.Plan.ResourceChanges {
			newRc := rc
			newRc.PlanSource = np.Name
			newRc.Address = prefix + rc.Address
			if newRc.PreviousAddress != "" {
				newRc.PreviousAddress = prefix + newRc.PreviousAddress
			}
			aggregated.ResourceChanges = append(aggregated.ResourceChanges, newRc)
		}

		// 2. Resource Drift
		for _, rd := range np.Plan.ResourceDrift {
			newRd := rd
			newRd.Address = prefix + rd.Address
			aggregated.ResourceDrift = append(aggregated.ResourceDrift, newRd)
		}

		// 3. Configuration
		if np.Plan.Configuration != nil {
			for _, res := range np.Plan.Configuration.RootModule.Resources {
				newRes := res
				newRes.Address = prefix + res.Address

				newDeps := make([]string, len(res.DependsOn))
				for i, d := range res.DependsOn {
					newDeps[i] = prefix + d
				}
				newRes.DependsOn = newDeps

				newExprs := make(map[string]Expression)
				for k, expr := range res.Expressions {
					newExpr := expr
					newRefs := make([]string, len(expr.References))
					for i, ref := range expr.References {
						newRefs[i] = prefix + ref
					}
					newExpr.References = newRefs
					newExprs[k] = newExpr
				}
				newRes.Expressions = newExprs

				aggregated.Configuration.RootModule.Resources = append(aggregated.Configuration.RootModule.Resources, newRes)
			}

			for callName, call := range np.Plan.Configuration.RootModule.ModuleCalls {
				childKey := fmt.Sprintf("%s.%s", np.Name, callName)
				aggregated.Configuration.RootModule.ModuleCalls[childKey] = qualifyModuleCall(call, prefix)
			}
		}

		// 4. Planned Values
		if np.Plan.PlannedValues != nil {
			qualifyStateModule(&np.Plan.PlannedValues.RootModule, prefix, &aggregated.PlannedValues.RootModule)
		}

		// 5. Prior State
		if np.Plan.PriorState != nil {
			root := getPriorStateRoot(np.Plan.PriorState)
			if root != nil {
				qualifyStateModule(root, prefix, aggregated.PriorState.RootModule)
			}
		}
	}

	return aggregated
}

func qualifyStateModule(src *StateModule, prefix string, dst *StateModule) {
	if src == nil || dst == nil {
		return
	}
	for _, res := range src.Resources {
		newRes := res
		newRes.Address = prefix + res.Address
		dst.Resources = append(dst.Resources, newRes)
	}
	for _, child := range src.ChildModules {
		qualifyStateModule(&child, prefix, dst)
	}
}

func qualifyModuleCall(call ModuleCall, prefix string) ModuleCall {
	qc := call
	newDeps := make([]string, len(call.DependsOn))
	for i, d := range call.DependsOn {
		newDeps[i] = prefix + d
	}
	qc.DependsOn = newDeps
	if call.Module != nil {
		qMod := qualifyConfigModule(call.Module, prefix)
		qc.Module = &qMod
	}
	return qc
}

func qualifyConfigModule(mod *ConfigModule, prefix string) ConfigModule {
	qm := ConfigModule{
		Resources:   make([]ConfigResource, 0, len(mod.Resources)),
		ModuleCalls: make(map[string]ModuleCall),
	}
	for _, res := range mod.Resources {
		newRes := res
		newRes.Address = prefix + res.Address
		qm.Resources = append(qm.Resources, newRes)
	}
	for k, c := range mod.ModuleCalls {
		qm.ModuleCalls[k] = qualifyModuleCall(c, prefix)
	}
	return qm
}
