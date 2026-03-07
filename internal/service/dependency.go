package service

import (
	"fmt"
	"strings"
)

type DependencyGraph struct {
	nodes map[string][]string // node -> dependencies
}

func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string][]string),
	}
}

func (g *DependencyGraph) AddNode(name string, dependencies []string) {
	g.nodes[name] = dependencies
}

func (g *DependencyGraph) Resolve() ([]string, error) {
	inDegree := make(map[string]int)
	dependents := make(map[string][]string) // node -> nodes that depend on it

	for name := range g.nodes {
		if _, ok := inDegree[name]; !ok {
			inDegree[name] = 0
		}
	}

	for name, deps := range g.nodes {
		for _, dep := range deps {
			if _, ok := g.nodes[dep]; ok {
				inDegree[name]++
				dependents[dep] = append(dependents[dep], name)
			}
		}
	}

	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var order []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)

		for _, dep := range dependents[node] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(order) != len(g.nodes) {
		var unresolved []string
		for name := range g.nodes {
			found := false
			for _, o := range order {
				if o == name {
					found = true
					break
				}
			}
			if !found {
				unresolved = append(unresolved, name)
			}
		}
		return nil, fmt.Errorf("dependency cycle detected involving: %s", strings.Join(unresolved, ", "))
	}

	return order, nil
}

func (g *DependencyGraph) ReverseOrder() ([]string, error) {
	order, err := g.Resolve()
	if err != nil {
		return nil, err
	}

	reversed := make([]string, len(order))
	for i, name := range order {
		reversed[len(order)-1-i] = name
	}
	return reversed, nil
}

func (g *DependencyGraph) IndependentGroups() ([][]string, error) {
	order, err := g.Resolve()
	if err != nil {
		return nil, err
	}

	started := make(map[string]bool)
	var groups [][]string

	for len(started) < len(order) {
		var group []string
		for _, name := range order {
			if started[name] {
				continue
			}
			allDepsStarted := true
			for _, dep := range g.nodes[name] {
				if _, inGraph := g.nodes[dep]; inGraph && !started[dep] {
					allDepsStarted = false
					break
				}
			}
			if allDepsStarted {
				group = append(group, name)
			}
		}
		if len(group) == 0 {
			break // safety valve
		}
		groups = append(groups, group)
		for _, name := range group {
			started[name] = true
		}
	}

	return groups, nil
}
