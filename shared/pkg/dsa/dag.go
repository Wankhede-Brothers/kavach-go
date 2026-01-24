// Package dsa provides data structures and algorithms.
// dag.go: Directed Acyclic Graph for task dependency tracking.
//
// Inspired by Beads (steveyegge/beads) - zero external dependencies.
// Reference: https://github.com/heimdalr/dag
package dsa

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Common errors for DAG operations
var (
	ErrVertexNotFound = errors.New("vertex not found")
	ErrEdgeNotFound   = errors.New("edge not found")
	ErrCycleDetected  = errors.New("adding edge would create cycle")
	ErrDuplicateEdge  = errors.New("edge already exists")
	ErrSelfLoop       = errors.New("self-loops not allowed")
)

// Vertex represents a node in the DAG (a task)
type Vertex struct {
	ID        string            `json:"id"`
	Label     string            `json:"label"`
	Status    string            `json:"status"` // pending, in_progress, completed, blocked
	Priority  int               `json:"priority"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	inDegree  int               // Number of incoming edges (blockers)
	outDegree int               // Number of outgoing edges (blocks)
}

// Edge represents a dependency between tasks
type Edge struct {
	From     string `json:"from"`     // Blocker task ID
	To       string `json:"to"`       // Blocked task ID
	Relation string `json:"relation"` // blocks, related, parent
}

// DAG is a thread-safe directed acyclic graph for task dependencies
type DAG struct {
	mu       sync.RWMutex
	vertices map[string]*Vertex
	edges    map[string]map[string]*Edge // from -> to -> edge
	reverse  map[string]map[string]*Edge // to -> from -> edge (for ancestor lookup)

	// Caches (invalidated on mutation)
	ancestorCache   map[string][]string
	descendantCache map[string][]string
	cacheValid      bool
}

// NewDAG creates a new empty DAG
func NewDAG() *DAG {
	return &DAG{
		vertices:        make(map[string]*Vertex),
		edges:           make(map[string]map[string]*Edge),
		reverse:         make(map[string]map[string]*Edge),
		ancestorCache:   make(map[string][]string),
		descendantCache: make(map[string][]string),
		cacheValid:      false,
	}
}

// GenerateID creates a Beads-style hash ID (kv-a1b2c3)
func GenerateID(label string) string {
	data := fmt.Sprintf("%s-%d", label, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "kv-" + hex.EncodeToString(hash[:])[:6]
}

// AddVertex adds a new task to the DAG
func (d *DAG) AddVertex(id, label string, priority int) (*Vertex, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.vertices[id]; exists {
		return d.vertices[id], nil // Idempotent
	}

	now := time.Now()
	v := &Vertex{
		ID:        id,
		Label:     label,
		Status:    "pending",
		Priority:  priority,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]string),
	}

	d.vertices[id] = v
	d.edges[id] = make(map[string]*Edge)
	d.reverse[id] = make(map[string]*Edge)
	d.invalidateCache()

	return v, nil
}

// AddEdge adds a dependency: from blocks to
func (d *DAG) AddEdge(from, to, relation string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Validate vertices exist
	if _, exists := d.vertices[from]; !exists {
		return fmt.Errorf("%w: %s", ErrVertexNotFound, from)
	}
	if _, exists := d.vertices[to]; !exists {
		return fmt.Errorf("%w: %s", ErrVertexNotFound, to)
	}

	// No self-loops
	if from == to {
		return ErrSelfLoop
	}

	// Check for duplicate
	if _, exists := d.edges[from][to]; exists {
		return ErrDuplicateEdge
	}

	// Check for cycle (would adding this edge create a path from 'to' back to 'from'?)
	if d.hasPathUnsafe(to, from) {
		return ErrCycleDetected
	}

	// Add edge
	edge := &Edge{From: from, To: to, Relation: relation}
	d.edges[from][to] = edge
	d.reverse[to][from] = edge

	// Update degrees
	d.vertices[from].outDegree++
	d.vertices[to].inDegree++

	// Update blocked status
	if d.vertices[to].Status == "pending" {
		d.vertices[to].Status = "blocked"
	}

	d.invalidateCache()
	return nil
}

// RemoveEdge removes a dependency
func (d *DAG) RemoveEdge(from, to string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.edges[from]; !exists {
		return ErrEdgeNotFound
	}
	if _, exists := d.edges[from][to]; !exists {
		return ErrEdgeNotFound
	}

	delete(d.edges[from], to)
	delete(d.reverse[to], from)

	d.vertices[from].outDegree--
	d.vertices[to].inDegree--

	// Update status if no more blockers
	if d.vertices[to].inDegree == 0 && d.vertices[to].Status == "blocked" {
		d.vertices[to].Status = "pending"
	}

	d.invalidateCache()
	return nil
}

// GetVertex returns a vertex by ID
func (d *DAG) GetVertex(id string) (*Vertex, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	v, exists := d.vertices[id]
	return v, exists
}

// UpdateStatus updates a task's status
func (d *DAG) UpdateStatus(id, status string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	v, exists := d.vertices[id]
	if !exists {
		return fmt.Errorf("%w: %s", ErrVertexNotFound, id)
	}

	v.Status = status
	v.UpdatedAt = time.Now()

	// If completed, check if any blocked tasks can be unblocked
	if status == "completed" {
		for to := range d.edges[id] {
			d.checkUnblock(to)
		}
	}

	return nil
}

// checkUnblock checks if a task can be unblocked (all blockers completed)
func (d *DAG) checkUnblock(id string) {
	v := d.vertices[id]
	if v.Status != "blocked" {
		return
	}

	// Check all incoming edges (blockers)
	for from := range d.reverse[id] {
		if d.vertices[from].Status != "completed" {
			return // Still blocked
		}
	}

	// All blockers completed
	v.Status = "pending"
	v.UpdatedAt = time.Now()
}

// Ready returns tasks with no incomplete blockers (Beads-style `bd ready`)
func (d *DAG) Ready() []*Vertex {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var ready []*Vertex

	for _, v := range d.vertices {
		if v.Status == "completed" {
			continue
		}

		// Check if all blockers are completed
		allBlocked := true
		for from := range d.reverse[v.ID] {
			if d.vertices[from].Status != "completed" {
				allBlocked = false
				break
			}
		}

		if allBlocked || len(d.reverse[v.ID]) == 0 {
			if v.Status != "blocked" {
				ready = append(ready, v)
			}
		}
	}

	return ready
}

// Blockers returns all tasks blocking a given task
func (d *DAG) Blockers(id string) []*Vertex {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var blockers []*Vertex
	for from := range d.reverse[id] {
		if v, exists := d.vertices[from]; exists {
			if v.Status != "completed" {
				blockers = append(blockers, v)
			}
		}
	}
	return blockers
}

// Blocks returns all tasks blocked by a given task
func (d *DAG) Blocks(id string) []*Vertex {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var blocks []*Vertex
	for to := range d.edges[id] {
		if v, exists := d.vertices[to]; exists {
			blocks = append(blocks, v)
		}
	}
	return blocks
}

// hasPathUnsafe checks if path exists (caller must hold lock)
func (d *DAG) hasPathUnsafe(from, to string) bool {
	visited := make(map[string]bool)
	return d.dfs(from, to, visited)
}

// dfs performs depth-first search
func (d *DAG) dfs(current, target string, visited map[string]bool) bool {
	if current == target {
		return true
	}
	if visited[current] {
		return false
	}
	visited[current] = true

	for next := range d.edges[current] {
		if d.dfs(next, target, visited) {
			return true
		}
	}
	return false
}

// TopologicalSort returns vertices in topological order
func (d *DAG) TopologicalSort() ([]*Vertex, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Kahn's algorithm
	inDegree := make(map[string]int)
	for id, v := range d.vertices {
		inDegree[id] = v.inDegree
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []*Vertex
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		sorted = append(sorted, d.vertices[current])

		for to := range d.edges[current] {
			inDegree[to]--
			if inDegree[to] == 0 {
				queue = append(queue, to)
			}
		}
	}

	if len(sorted) != len(d.vertices) {
		return nil, ErrCycleDetected
	}

	return sorted, nil
}

// Stats returns DAG statistics
func (d *DAG) Stats() map[string]int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := map[string]int{
		"total":       len(d.vertices),
		"pending":     0,
		"in_progress": 0,
		"completed":   0,
		"blocked":     0,
		"edges":       0,
	}

	for _, v := range d.vertices {
		stats[v.Status]++
	}

	for _, edges := range d.edges {
		stats["edges"] += len(edges)
	}

	return stats
}

// invalidateCache invalidates ancestor/descendant caches
func (d *DAG) invalidateCache() {
	d.cacheValid = false
	d.ancestorCache = make(map[string][]string)
	d.descendantCache = make(map[string][]string)
}
