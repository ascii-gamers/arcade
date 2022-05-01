package arcade

import (
	"testing"
)

func TestBellmanFord(t *testing.T) {
	table := NewRoutingTable()
	table.Update("a", map[string]float64{"b": 1})
	table.Update("b", map[string]float64{"a": 1, "c": 2})
	table.Update("c", map[string]float64{"b": 2})

	if table.GetShortestPath("a", "b").Cost != 1 {
		t.Error("Expected 1, got", table.GetShortestPath("a", "b").Cost)
	}

	if table.GetShortestPath("a", "c").Cost != 3 {
		t.Error("Expected 3, got", table.GetShortestPath("a", "c").Cost)
	}
}
