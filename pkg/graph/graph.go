package graph

import (
	"sort"
	"strings"

	"github.com/smford/tf-blast/pkg/parser"
)

// Node represents a resource or module in the infrastructure graph.
type Node struct {
	Address string
	Type    string
	Name    string
	Module  string
	Change  *parser.ResourceChange
}

// Graph is a memory-efficient directed graph using adjacency lists.
// An edge from A to B means "A depends on B" (B is upstream, A is downstream).
type Graph struct {
	nodes      map[string]*Node
	upstream   map[string]map[string]bool // A -> set of B (A depends on B)
	downstream map[string]map[string]bool // B -> set of A (A depends on B, so B changes impact A)
}

// NewGraph creates a new empty Graph.
func NewGraph() *Graph {
	return &Graph{
		nodes:      make(map[string]*Node),
		upstream:   make(map[string]map[string]bool),
		downstream: make(map[string]map[string]bool),
	}
}

// AddNode registers a node in the graph.
func (g *Graph) AddNode(address string, nodeType, name, module string, change *parser.ResourceChange) *Node {
	cleanAddr := parser.StripIndex(address)
	if existing, found := g.nodes[cleanAddr]; found {
		if change != nil && existing.Change == nil {
			existing.Change = change
		}
		if nodeType != "" && existing.Type == "" {
			existing.Type = nodeType
		}
		return existing
	}

	node := &Node{
		Address: cleanAddr,
		Type:    nodeType,
		Name:    name,
		Module:  module,
		Change:  change,
	}
	g.nodes[cleanAddr] = node
	return node
}

// GetNode returns the node by address, or nil if not found.
func (g *Graph) GetNode(address string) *Node {
	cleanAddr := parser.StripIndex(address)
	return g.nodes[cleanAddr]
}

// Nodes returns all nodes in deterministic address-sorted order.
func (g *Graph) Nodes() []*Node {
	list := make([]*Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		list = append(list, n)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Address < list[j].Address
	})
	return list
}

// AddEdge records that 'from' depends on 'to' (to is upstream, from is downstream).
func (g *Graph) AddEdge(from, to string) {
	from = parser.StripIndex(from)
	to = parser.StripIndex(to)

	if from == "" || to == "" || from == to {
		return
	}

	// Ensure both nodes exist
	if _, ok := g.nodes[from]; !ok {
		g.AddNode(from, extractTypeFromAddr(from), "", "", nil)
	}
	if _, ok := g.nodes[to]; !ok {
		g.AddNode(to, extractTypeFromAddr(to), "", "", nil)
	}

	if g.upstream[from] == nil {
		g.upstream[from] = make(map[string]bool)
	}
	g.upstream[from][to] = true

	if g.downstream[to] == nil {
		g.downstream[to] = make(map[string]bool)
	}
	g.downstream[to][from] = true
}

// GetDirectDownstream returns immediate downstream dependents of 'address'.
func (g *Graph) GetDirectDownstream(address string) []string {
	address = parser.StripIndex(address)
	edges := g.downstream[address]
	if len(edges) == 0 {
		return nil
	}
	result := make([]string, 0, len(edges))
	for target := range edges {
		result = append(result, target)
	}
	sort.Strings(result)
	return result
}

// GetDirectUpstream returns immediate upstream dependencies of 'address'.
func (g *Graph) GetDirectUpstream(address string) []string {
	address = parser.StripIndex(address)
	edges := g.upstream[address]
	if len(edges) == 0 {
		return nil
	}
	result := make([]string, 0, len(edges))
	for target := range edges {
		result = append(result, target)
	}
	sort.Strings(result)
	return result
}

// GetTransitiveDownstream returns all indirectly or directly affected resources
// using BFS to prevent recursion issues and avoid infinite loops on cycles.
func (g *Graph) GetTransitiveDownstream(root string) []string {
	root = parser.StripIndex(root)
	visited := make(map[string]bool)
	var queue []string

	visited[root] = true
	queue = append(queue, root)

	var result []string

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for next := range g.downstream[curr] {
			if !visited[next] {
				visited[next] = true
				result = append(result, next)
				queue = append(queue, next)
			}
		}
	}

	sort.Strings(result)
	return result
}

// GetTransitiveUpstream returns all upstream dependencies using BFS.
func (g *Graph) GetTransitiveUpstream(root string) []string {
	root = parser.StripIndex(root)
	visited := make(map[string]bool)
	var queue []string

	visited[root] = true
	queue = append(queue, root)

	var result []string

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for next := range g.upstream[curr] {
			if !visited[next] {
				visited[next] = true
				result = append(result, next)
				queue = append(queue, next)
			}
		}
	}

	sort.Strings(result)
	return result
}

// HasCycles checks if the dependency graph contains any circular dependencies.
func (g *Graph) HasCycles() bool {
	visited := make(map[string]int) // 0: unvisited, 1: visiting, 2: visited

	var dfs func(string) bool
	dfs = func(u string) bool {
		visited[u] = 1
		for v := range g.upstream[u] {
			if visited[v] == 1 {
				return true
			}
			if visited[v] == 0 {
				if dfs(v) {
					return true
				}
			}
		}
		visited[u] = 2
		return false
	}

	for addr := range g.nodes {
		if visited[addr] == 0 {
			if dfs(addr) {
				return true
			}
		}
	}
	return false
}

// NodeCount returns the total number of registered nodes.
func (g *Graph) NodeCount() int {
	return len(g.nodes)
}

// EdgeCount returns the total number of directed edges.
func (g *Graph) EdgeCount() int {
	count := 0
	for _, targets := range g.upstream {
		count += len(targets)
	}
	return count
}

func extractTypeFromAddr(addr string) string {
	parts := strings.Split(addr, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return ""
}
