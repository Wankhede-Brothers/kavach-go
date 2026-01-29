package dag

import (
	"os"
	"testing"
)

// TestTopoLevels: 5 nodes, 4 edges → verify correct parallel groups.
//
//	A ──→ C ──→ E
//	B ──→ D ──↗
//
// Level 0: A, B (parallel research)
// Level 1: C, D (parallel, depend on A/B respectively)
// Level 2: E (depends on C and D)
func TestTopoLevels(t *testing.T) {
	state := NewDAGState("test-session", "build payment handler")

	nodes := []*Node{
		{ID: "a", Subject: "Research webhook patterns", Agent: "research"},
		{ID: "b", Subject: "Research Hyperswitch API", Agent: "research"},
		{ID: "c", Subject: "Implement webhook handler", Agent: "backend"},
		{ID: "d", Subject: "Implement payment types", Agent: "backend"},
		{ID: "e", Subject: "Integration test", Agent: "testing"},
	}
	for _, n := range nodes {
		if err := state.AddNode(n); err != nil {
			t.Fatalf("AddNode(%s): %v", n.ID, err)
		}
	}

	edges := [][2]string{{"a", "c"}, {"b", "d"}, {"c", "e"}, {"d", "e"}}
	for _, e := range edges {
		if err := state.AddEdge(e[0], e[1]); err != nil {
			t.Fatalf("AddEdge(%s→%s): %v", e[0], e[1], err)
		}
	}

	levels, err := TopoLevels(state)
	if err != nil {
		t.Fatalf("TopoLevels: %v", err)
	}

	if len(levels) != 3 {
		t.Fatalf("expected 3 levels, got %d", len(levels))
	}
	if len(levels[0].Nodes) != 2 {
		t.Errorf("level 0: expected 2 nodes, got %d", len(levels[0].Nodes))
	}
	if len(levels[1].Nodes) != 2 {
		t.Errorf("level 1: expected 2 nodes, got %d", len(levels[1].Nodes))
	}
	if len(levels[2].Nodes) != 1 {
		t.Errorf("level 2: expected 1 node, got %d", len(levels[2].Nodes))
	}
	if state.MaxLevel != 2 {
		t.Errorf("expected MaxLevel=2, got %d", state.MaxLevel)
	}
}

func TestCycleDetection(t *testing.T) {
	state := NewDAGState("test-cycle", "cycle test")
	state.AddNode(&Node{ID: "x", Subject: "X"})
	state.AddNode(&Node{ID: "y", Subject: "Y"})
	state.AddNode(&Node{ID: "z", Subject: "Z"})

	_ = state.AddEdge("x", "y")
	_ = state.AddEdge("y", "z")
	err := state.AddEdge("z", "x") // creates cycle
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
}

func TestNodeStatusPropagation(t *testing.T) {
	state := NewDAGState("test-prop", "propagation test")
	state.AddNode(&Node{ID: "a", Subject: "A", Status: StatusReady})
	state.AddNode(&Node{ID: "b", Subject: "B", Status: StatusPending})
	state.AddNode(&Node{ID: "c", Subject: "C", Status: StatusPending})
	_ = state.AddEdge("a", "b")
	_ = state.AddEdge("b", "c")

	// Complete A → B should become ready
	state.UpdateNodeStatus("a", StatusDone)
	if state.Nodes["b"].Status != StatusReady {
		t.Errorf("expected b=ready, got %s", state.Nodes["b"].Status)
	}

	// Fail B → C should become skipped
	state.UpdateNodeStatus("b", StatusFailed)
	if state.Nodes["c"].Status != StatusSkipped {
		t.Errorf("expected c=skipped, got %s", state.Nodes["c"].Status)
	}
	if !state.IsComplete() {
		t.Error("expected DAG complete after all terminal")
	}
}

func TestScheduleResearchParallel(t *testing.T) {
	breakdown := []string{
		"Research webhook patterns",
		"Research Hyperswitch API",
		"Implement handler",
		"Write tests",
	}
	agents := []string{"research", "research", "backend", "testing"}
	nodes := Decompose(breakdown, agents)

	state, err := Schedule("test-sched", "build webhook", nodes)
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	// Research nodes should be level 0 (parallel), impl level 1, tests level 2
	researchCount := 0
	for _, n := range state.Nodes {
		if n.Level == 0 {
			researchCount++
		}
	}
	if researchCount < 2 {
		t.Errorf("expected at least 2 research nodes at level 0, got %d", researchCount)
	}
}

func TestStatePersistence(t *testing.T) {
	sid := "test-persist-roundtrip"
	state := NewDAGState(sid, "persist test")
	state.AddNode(&Node{ID: "p1", Subject: "Persist node", Agent: "test"})

	if err := Save(state); err != nil {
		t.Fatalf("Save: %v", err)
	}
	defer Delete(sid)

	loaded, err := Load(sid)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.ID != state.ID {
		t.Errorf("ID mismatch: %s != %s", loaded.ID, state.ID)
	}
	if len(loaded.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(loaded.Nodes))
	}
	if _, ok := loaded.Nodes["p1"]; !ok {
		t.Error("node p1 not found after load")
	}
}

func TestBuildDirective(t *testing.T) {
	state := NewDAGState("test-dir", "directive test")
	state.AddNode(&Node{ID: "d1", Subject: "Task 1", Agent: "eng", Status: StatusReady, Level: 0})
	state.AddNode(&Node{ID: "d2", Subject: "Task 2", Agent: "eng", Status: StatusReady, Level: 0})
	state.MaxLevel = 1

	directive := BuildDirective(state)
	if directive == "" {
		t.Fatal("expected non-empty directive")
	}
	if !contains(directive, "PARALLEL_DISPATCH") {
		t.Error("directive missing PARALLEL_DISPATCH")
	}
	if !contains(directive, "count: 2") {
		t.Error("directive missing count: 2")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
