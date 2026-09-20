package graph

import (
	"strings"

	"github.com/smford/tf-blast/pkg/parser"
)

// BuildGraph constructs an in-memory dependency graph from a parsed Terraform plan.
func BuildGraph(plan *parser.Plan) *Graph {
	g := NewGraph()
	if plan == nil {
		return g
	}

	// 1. Add all resources from resource_changes
	for i := range plan.ResourceChanges {
		rc := &plan.ResourceChanges[i]
		cleanAddr := parser.StripIndex(rc.Address)
		g.AddNode(cleanAddr, rc.Type, rc.Name, rc.ModuleAddress, rc)
	}

	// 2. Add any configuration resources not already in resource_changes
	if plan.Configuration != nil {
		registerConfigModuleResources(g, &plan.Configuration.RootModule, "")
	}

	// 3. Extract and add configuration dependencies (depends_on and references)
	configRefs := parser.ExtractConfigDependencies(plan)
	for _, ref := range configRefs {
		addResolvedDependency(g, ref.Source, ref.Target)
	}

	// 4. Extract and add state-inferred relationships
	stateRefs := parser.ExtractStateRelationships(plan)
	for _, ref := range stateRefs {
		addResolvedDependency(g, ref.Source, ref.Target)
	}

	return g
}

func registerConfigModuleResources(g *Graph, mod *parser.ConfigModule, prefix string) {
	if mod == nil {
		return
	}
	for _, res := range mod.Resources {
		addr := qualifyAddress(prefix, res.Address)
		g.AddNode(addr, res.Type, res.Name, prefix, nil)
	}
	for callName, call := range mod.ModuleCalls {
		childPrefix := callName
		if prefix != "" {
			childPrefix = prefix + ".module." + callName
		} else {
			childPrefix = "module." + callName
		}
		if call.Module != nil {
			registerConfigModuleResources(g, call.Module, childPrefix)
		}
	}
}

func qualifyAddress(prefix, address string) string {
	cleanAddr := strings.TrimSpace(address)
	if cleanAddr == "" {
		return ""
	}
	if strings.HasPrefix(cleanAddr, "module.") || prefix == "" {
		return cleanAddr
	}
	return prefix + "." + cleanAddr
}

// addResolvedDependency links source to target. If target is a module (e.g. module.vpc),
// it creates edges from source to all resources inside that module.
func addResolvedDependency(g *Graph, source, target string) {
	source = parser.StripIndex(source)
	target = parser.StripIndex(target)

	if source == "" || target == "" || source == target {
		return
	}

	// If target is a module prefix like "module.vpc"
	if strings.HasPrefix(target, "module.") && !hasResourceType(target) {
		matched := false
		for _, node := range g.Nodes() {
			if strings.HasPrefix(node.Address, target+".") {
				g.AddEdge(source, node.Address)
				matched = true
			}
		}
		if !matched {
			g.AddEdge(source, target)
		}
		return
	}

	g.AddEdge(source, target)
}

func hasResourceType(addr string) bool {
	// e.g. "module.vpc.aws_subnet.private" has 4 parts, 3rd is resource type "aws_subnet"
	// "module.vpc" has 2 parts, no resource type
	parts := strings.Split(addr, ".")
	if len(parts) >= 2 {
		last := parts[len(parts)-1]
		secondLast := parts[len(parts)-2]
		if strings.Contains(secondLast, "_") || strings.Contains(last, "_") {
			return true
		}
	}
	return false
}
