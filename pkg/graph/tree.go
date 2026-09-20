package graph

import (
	"strings"

	"github.com/smford/tf-blast/pkg/parser"
)

// TreeNode represents a node in the cascading blast radius tree.
type TreeNode struct {
	Address  string
	Type     string
	Action   string
	Severity string
	Children []*TreeNode
}

// BuildDownstreamTree builds a hierarchical tree of downstream impacted resources starting from root.
// maxDepth prevents runaway trees in highly connected graphs.
func (g *Graph) BuildDownstreamTree(root string, maxDepth int) *TreeNode {
	root = parser.StripIndex(root)
	visited := make(map[string]bool)

	var build func(addr string, depth int) *TreeNode
	build = func(addr string, depth int) *TreeNode {
		visited[addr] = true

		node := g.GetNode(addr)
		action := ""
		nodeType := ""
		if node != nil {
			nodeType = node.Type
			if node.Change != nil && len(node.Change.Change.Actions) > 0 {
				action = strings.Join(node.Change.Change.Actions, ",")
			}
		}

		tNode := &TreeNode{
			Address: addr,
			Type:    nodeType,
			Action:  action,
		}

		if depth >= maxDepth {
			return tNode
		}

		downstream := g.GetDirectDownstream(addr)
		for _, ds := range downstream {
			if !visited[ds] {
				child := build(ds, depth+1)
				tNode.Children = append(tNode.Children, child)
			}
		}

		return tNode
	}

	return build(root, 0)
}

// RenderTree renders a TreeNode into an ASCII box-drawing tree representation.
func (t *TreeNode) RenderTree() string {
	if t == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(t.Address)
	if t.Action != "" {
		sb.WriteString(" (")
		sb.WriteString(t.Action)
		sb.WriteString(")")
	}
	sb.WriteString("\n")

	renderChildren(&sb, t.Children, "")
	return sb.String()
}

func renderChildren(sb *strings.Builder, children []*TreeNode, prefix string) {
	count := len(children)
	for i, child := range children {
		isLast := i == count-1
		marker := "├── "
		childPrefix := prefix + "│   "
		if isLast {
			marker = "└── "
			childPrefix = prefix + "    "
		}

		sb.WriteString(prefix)
		sb.WriteString(marker)
		sb.WriteString(child.Address)
		if child.Action != "" {
			sb.WriteString(" (")
			sb.WriteString(child.Action)
			sb.WriteString(")")
		} else {
			sb.WriteString(" (downstream)")
		}
		sb.WriteString("\n")

		renderChildren(sb, child.Children, childPrefix)
	}
}
