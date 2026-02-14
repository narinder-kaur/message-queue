package queue

import (
	"testing"
)

// TestMemoryQueue tests basic MemoryQueue creation
func TestMemoryQueue(t *testing.T) {
	q := &MemoryQueue{}
	if q == nil {
		t.Error("expected non-nil MemoryQueue")
	}
}
