package service

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildGraph(nodes map[string][]string) *DependencyGraph {
	g := NewDependencyGraph()
	for name, deps := range nodes {
		g.AddNode(name, deps)
	}
	return g
}

func comesAfter(order []string, first, second string) bool {
	fi, si := -1, -1
	for i, name := range order {
		if name == first {
			fi = i
		}
		if name == second {
			si = i
		}
	}
	return fi >= 0 && si >= 0 && fi < si
}

func TestNewDependencyGraph_IsEmpty(t *testing.T) {
	g := NewDependencyGraph()
	require.NotNil(t, g)

	order, err := g.Resolve()
	require.NoError(t, err)
	assert.Empty(t, order)
}

func TestAddNode_SingleNode(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("alpha", nil)

	order, err := g.Resolve()
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha"}, order)
}

func TestAddNode_OverwritesDependencies(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("alpha", []string{"beta"})
	g.AddNode("beta", nil)
	g.AddNode("alpha", nil)

	order, err := g.Resolve()
	require.NoError(t, err)
	assert.Len(t, order, 2)
}

func TestResolve_LinearChain(t *testing.T) {
	g := buildGraph(map[string][]string{
		"gamma": {},
		"beta":  {"gamma"},
		"alpha": {"beta"},
	})

	order, err := g.Resolve()
	require.NoError(t, err)
	require.Len(t, order, 3)

	assert.True(t, comesAfter(order, "gamma", "beta"), "gamma should come before beta")
	assert.True(t, comesAfter(order, "beta", "alpha"), "beta should come before alpha")
}

func TestResolve_NoDependencies(t *testing.T) {
	g := buildGraph(map[string][]string{
		"alpha": {},
		"beta":  {},
		"gamma": {},
	})

	order, err := g.Resolve()
	require.NoError(t, err)
	assert.Len(t, order, 3)
}

func TestResolve_DiamondDependency(t *testing.T) {
	g := buildGraph(map[string][]string{
		"D": {},
		"B": {"D"},
		"C": {"D"},
		"A": {"B", "C"},
	})

	order, err := g.Resolve()
	require.NoError(t, err)
	require.Len(t, order, 4)

	assert.True(t, comesAfter(order, "D", "B"), "D should come before B")
	assert.True(t, comesAfter(order, "D", "C"), "D should come before C")
	assert.True(t, comesAfter(order, "B", "A"), "B should come before A")
	assert.True(t, comesAfter(order, "C", "A"), "C should come before A")
}

func TestResolve_ContainsAllNodes(t *testing.T) {
	nodes := map[string][]string{
		"svc1": {},
		"svc2": {"svc1"},
		"svc3": {"svc1"},
		"svc4": {"svc2", "svc3"},
	}
	g := buildGraph(nodes)

	order, err := g.Resolve()
	require.NoError(t, err)

	assert.Len(t, order, len(nodes))

	seen := make(map[string]int)
	for _, name := range order {
		seen[name]++
	}
	for name := range nodes {
		assert.Equal(t, 1, seen[name], "expected %s exactly once", name)
	}
}

func TestResolve_ExternalDependencyIgnored(t *testing.T) {
	g := buildGraph(map[string][]string{
		"alpha": {"external"},
	})

	order, err := g.Resolve()
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha"}, order)
}

func TestResolve_DirectCycle_ReturnsError(t *testing.T) {
	g := buildGraph(map[string][]string{
		"A": {"B"},
		"B": {"A"},
	})

	_, err := g.Resolve()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestResolve_ThreeNodeCycle_ReturnsError(t *testing.T) {
	g := buildGraph(map[string][]string{
		"A": {"B"},
		"B": {"C"},
		"C": {"A"},
	})

	_, err := g.Resolve()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestResolve_SelfLoop_ReturnsError(t *testing.T) {
	g := buildGraph(map[string][]string{
		"A": {"A"},
	})

	_, err := g.Resolve()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestResolve_PartialCycleWithValidNodes_ReturnsError(t *testing.T) {
	g := buildGraph(map[string][]string{
		"root":  {},
		"nodeA": {"root", "nodeB"},
		"nodeB": {"nodeA"},
	})

	_, err := g.Resolve()
	require.Error(t, err)
}

func TestReverseOrder_IsReversedTopological(t *testing.T) {
	g := buildGraph(map[string][]string{
		"db":   {},
		"api":  {"db"},
		"web":  {"api"},
	})

	forward, err := g.Resolve()
	require.NoError(t, err)

	reversed, err := g.ReverseOrder()
	require.NoError(t, err)

	require.Len(t, reversed, len(forward))
	for i, name := range forward {
		assert.Equal(t, name, reversed[len(forward)-1-i],
			"reversed[%d] should be forward[%d]", len(forward)-1-i, i)
	}
}

func TestReverseOrder_CycleReturnsError(t *testing.T) {
	g := buildGraph(map[string][]string{
		"A": {"B"},
		"B": {"A"},
	})

	_, err := g.ReverseOrder()
	require.Error(t, err)
}

func TestReverseOrder_SingleNode(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("only", nil)

	reversed, err := g.ReverseOrder()
	require.NoError(t, err)
	assert.Equal(t, []string{"only"}, reversed)
}

func TestIndependentGroups_NoDependencies_SingleGroup(t *testing.T) {
	g := buildGraph(map[string][]string{
		"A": {},
		"B": {},
		"C": {},
	})

	groups, err := g.IndependentGroups()
	require.NoError(t, err)
	require.Len(t, groups, 1, "all independent nodes should be in one group")
	assert.Len(t, groups[0], 3)
}

func TestIndependentGroups_LinearChain_OneNodePerGroup(t *testing.T) {
	g := buildGraph(map[string][]string{
		"A": {},
		"B": {"A"},
		"C": {"B"},
	})

	groups, err := g.IndependentGroups()
	require.NoError(t, err)
	assert.Len(t, groups, 3, "linear chain should produce one group per level")

	assert.Equal(t, []string{"A"}, groups[0])
	assert.Equal(t, []string{"B"}, groups[1])
	assert.Equal(t, []string{"C"}, groups[2])
}

func TestIndependentGroups_DiamondDependency_ThreeGroups(t *testing.T) {
	g := buildGraph(map[string][]string{
		"D": {},
		"B": {"D"},
		"C": {"D"},
		"A": {"B", "C"},
	})

	groups, err := g.IndependentGroups()
	require.NoError(t, err)
	require.Len(t, groups, 3)

	assert.Equal(t, []string{"D"}, groups[0])

	group1Names := make([]string, len(groups[1]))
	copy(group1Names, groups[1])
	sort.Strings(group1Names)
	assert.Equal(t, []string{"B", "C"}, group1Names)

	assert.Equal(t, []string{"A"}, groups[2])
}

func TestIndependentGroups_TwoIndependentChains(t *testing.T) {
	g := buildGraph(map[string][]string{
		"X": {},
		"Y": {"X"},
		"P": {},
		"Q": {"P"},
	})

	groups, err := g.IndependentGroups()
	require.NoError(t, err)
	require.Len(t, groups, 2)

	group0 := make([]string, len(groups[0]))
	copy(group0, groups[0])
	sort.Strings(group0)
	assert.Equal(t, []string{"P", "X"}, group0)

	group1 := make([]string, len(groups[1]))
	copy(group1, groups[1])
	sort.Strings(group1)
	assert.Equal(t, []string{"Q", "Y"}, group1)
}

func TestIndependentGroups_EmptyGraph(t *testing.T) {
	g := NewDependencyGraph()
	groups, err := g.IndependentGroups()
	require.NoError(t, err)
	assert.Empty(t, groups)
}

func TestIndependentGroups_CycleReturnsError(t *testing.T) {
	g := buildGraph(map[string][]string{
		"A": {"B"},
		"B": {"A"},
	})

	_, err := g.IndependentGroups()
	require.Error(t, err)
}

func TestIndependentGroups_AllNodesPresent(t *testing.T) {
	nodes := map[string][]string{
		"db":      {},
		"cache":   {},
		"api":     {"db", "cache"},
		"worker":  {"db"},
		"frontend": {"api"},
	}
	g := buildGraph(nodes)

	groups, err := g.IndependentGroups()
	require.NoError(t, err)

	seen := make(map[string]int)
	for _, group := range groups {
		for _, name := range group {
			seen[name]++
		}
	}
	for name := range nodes {
		assert.Equal(t, 1, seen[name], "node %q should appear exactly once", name)
	}
}

func TestIndependentGroups_OrderWithinGroupsIsValid(t *testing.T) {
	nodes := map[string][]string{
		"alpha": {},
		"beta":  {"alpha"},
		"gamma": {"alpha"},
		"delta": {"beta", "gamma"},
	}
	g := buildGraph(nodes)

	groups, err := g.IndependentGroups()
	require.NoError(t, err)

	var flat []string
	for _, group := range groups {
		flat = append(flat, group...)
	}

	pos := make(map[string]int)
	for i, name := range flat {
		pos[name] = i
	}
	for name, deps := range nodes {
		for _, dep := range deps {
			if _, inGraph := nodes[dep]; inGraph {
				assert.Less(t, pos[dep], pos[name],
					"dependency %q should come before %q in flattened order", dep, name)
			}
		}
	}
}
