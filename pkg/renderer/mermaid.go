package renderer

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/smford/tf-blast/pkg/analyzer"
	"github.com/smford/tf-blast/pkg/graph"
)

var idSanitizerRegex = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// RenderMermaid outputs a visual Mermaid.js flowchart representation of the blast radius DAG.
func RenderMermaid(w io.Writer, report *analyzer.AnalysisReport) error {
	if report == nil {
		return nil
	}

	fmt.Fprintln(w, "```mermaid")
	fmt.Fprintln(w, "graph TD")
	fmt.Fprintln(w, "  %% Styles for blast radius severity")
	fmt.Fprintln(w, "  classDef critical fill:#e74c3c,stroke:#c0392b,color:#ffffff,stroke-width:2px;")
	fmt.Fprintln(w, "  classDef high fill:#e67e22,stroke:#d35400,color:#ffffff,stroke-width:2px;")
	fmt.Fprintln(w, "  classDef medium fill:#f39c12,stroke:#d68910,color:#ffffff,stroke-width:2px;")
	fmt.Fprintln(w, "  classDef low fill:#27ae60,stroke:#1e8449,color:#ffffff,stroke-width:2px;")
	fmt.Fprintln(w, "  classDef downstream fill:#34495e,stroke:#2c3e50,color:#ffffff,stroke-dasharray: 4 4;")
	fmt.Fprintln(w)

	seenNodes := make(map[string]bool)
	seenEdges := make(map[string]bool)

	for _, res := range report.Resources {
		if res.DownstreamTree != nil && len(res.DownstreamTree.Children) > 0 {
			renderMermaidSubtree(w, res.DownstreamTree, res.Severity, seenNodes, seenEdges)
		}
	}

	if len(seenNodes) == 0 {
		fmt.Fprintln(w, "  no_blast[\"No Cascading Blast Radius Detected\"]:::low")
	}

	fmt.Fprintln(w, "```")
	return nil
}

func renderMermaidSubtree(
	w io.Writer,
	node *graph.TreeNode,
	rootSev analyzer.Severity,
	seenNodes map[string]bool,
	seenEdges map[string]bool,
) {
	if node == nil {
		return
	}

	rootID := sanitizeMermaidID(node.Address)
	if !seenNodes[rootID] {
		seenNodes[rootID] = true
		styleClass := severityToMermaidClass(rootSev)
		actionLabel := ""
		if node.Action != "" {
			actionLabel = fmt.Sprintf(" (%s)", strings.ToUpper(node.Action))
		}
		fmt.Fprintf(w, "  %s[\"%s%s\"]:::%s\n", rootID, node.Address, actionLabel, styleClass)
	}

	for _, child := range node.Children {
		childID := sanitizeMermaidID(child.Address)
		if !seenNodes[childID] {
			seenNodes[childID] = true
			fmt.Fprintf(w, "  %s[\"%s\"]:::downstream\n", childID, child.Address)
		}

		edgeKey := rootID + "-->" + childID
		if !seenEdges[edgeKey] {
			seenEdges[edgeKey] = true
			fmt.Fprintf(w, "  %s --> %s\n", rootID, childID)
		}

		// Recurse for deeper children
		renderMermaidSubtree(w, child, analyzer.SeverityLow, seenNodes, seenEdges)
	}
}

func sanitizeMermaidID(addr string) string {
	clean := idSanitizerRegex.ReplaceAllString(addr, "_")
	return "n_" + clean
}

func severityToMermaidClass(s analyzer.Severity) string {
	switch s {
	case analyzer.SeverityCritical:
		return "critical"
	case analyzer.SeverityHigh:
		return "high"
	case analyzer.SeverityMedium:
		return "medium"
	default:
		return "low"
	}
}
