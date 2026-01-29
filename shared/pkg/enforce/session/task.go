// Package session provides session state management.
// task.go: Task state management functions.
// DACE: Single responsibility - task tracking only.
//
// Dead code audit: SetTask, AddFileModified, ClearTask, HasTask were never called.
// Task state is managed by the task gate (gates/task.go) via TaskCreate/TaskUpdate hooks.
// File retained as placeholder for future task management functions.
package session
