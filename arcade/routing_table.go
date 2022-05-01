package arcade

import (
	"math"
	"sync"

	"golang.org/x/exp/slices"
)

type RoutingPath struct {
	Path []string
	Cost float64
}

type RoutingTable struct {
	sync.RWMutex

	table         map[string]map[string]float64
	shortestPaths map[string]map[string]*RoutingPath
}

func NewRoutingTable() *RoutingTable {
	return &RoutingTable{
		table:         make(map[string]map[string]float64),
		shortestPaths: make(map[string]map[string]*RoutingPath),
	}
}

func (t *RoutingTable) runBellmanFord() {
	for from := range t.table {
		size := len(t.table)
		distances := make(map[string]float64, size)
		predecessors := make(map[string][]string, size)

		for clientID := range t.table {
			distances[clientID] = math.MaxFloat64
		}

		distances[from] = 0

		for i := 0; i < size-1; i++ {
			for from := range t.table {
				for to := range t.table[from] {
					newDist := distances[from] + t.table[from][to]

					if newDist < distances[to] {
						distances[to] = newDist
						predecessors[to] = []string{from}
					} else if newDist == distances[to] {
						if !slices.Contains(predecessors[to], from) {
							predecessors[to] = append(predecessors[to], from)
						}
					}
				}
			}
		}

		if _, ok := t.shortestPaths[from]; !ok {
			t.shortestPaths[from] = make(map[string]*RoutingPath)
		}

		for to := range t.table {
			t.shortestPaths[from][to] = &RoutingPath{
				Path: predecessors[to],
				Cost: distances[to],
			}
		}
	}
}

func (t *RoutingTable) Update(clientID string, adjacency map[string]float64) {
	t.Lock()
	defer t.Unlock()

	for k, v := range adjacency {
		if _, ok := t.table[clientID]; !ok {
			t.table[clientID] = make(map[string]float64)
		}

		if _, ok := t.table[k]; !ok {
			t.table[k] = make(map[string]float64)
		}

		t.table[clientID][k] = v
		t.table[k][clientID] = v
	}

	// Naive implementation for now: recalculate all shortest paths on every update
	// TODO: optimize
	t.runBellmanFord()
}

func (t *RoutingTable) GetShortestPath(from, to string) *RoutingPath {
	t.RLock()
	defer t.RUnlock()

	if _, ok := t.shortestPaths[from]; !ok {
		return nil
	}

	return t.shortestPaths[from][to]
}

func (t *RoutingTable) GetNetworkTopology() map[string]map[string]float64 {
	t.RLock()
	defer t.RUnlock()

	return t.table
}
