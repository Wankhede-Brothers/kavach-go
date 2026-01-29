// Package dag provides a parallel DAG scheduler for Kavach orchestration.
// topo.go: Kahn's algorithm for topological level assignment.
package dag

import "fmt"

// TopoLevels groups nodes into parallel execution waves using Kahn's algorithm.
// Sets node.Level and state.MaxLevel. Returns error on cycle.
func TopoLevels(state *DAGState) ([]ParallelLevel, error) {
	inDeg := make(map[string]int, len(state.Nodes))
	for id, n := range state.Nodes {
		inDeg[id] = len(n.DependsOn)
	}

	// Seed queue with zero in-degree nodes
	var queue []string
	for id, deg := range inDeg {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var levels []ParallelLevel
	processed := 0

	for len(queue) > 0 {
		level := ParallelLevel{Level: len(levels)}
		var nextQueue []string

		for _, id := range queue {
			node := state.Nodes[id]
			node.Level = level.Level
			level.Nodes = append(level.Nodes, node)
			processed++

			for _, blockedID := range node.Blocks {
				inDeg[blockedID]--
				if inDeg[blockedID] == 0 {
					nextQueue = append(nextQueue, blockedID)
				}
			}
		}

		levels = append(levels, level)
		queue = nextQueue
	}

	if processed != len(state.Nodes) {
		return nil, fmt.Errorf("cycle detected: processed %d of %d nodes", processed, len(state.Nodes))
	}

	state.MaxLevel = len(levels) - 1
	return levels, nil
}
