package ast

// PQHeap is a persistent leftist min-heap keyed by an integer priority. All
// operations are immutable and O(log n); older versions stay valid, matching
// the persistent maps/sets/vectors.
type PQHeap struct {
	Prio  int64
	Val   Node
	Left  *PQHeap
	Right *PQHeap
	Rank  int
}

func pqRank(h *PQHeap) int {
	if h == nil {
		return 0
	}
	return h.Rank
}

func pqMerge(a, b *PQHeap) *PQHeap {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if a.Prio > b.Prio {
		a, b = b, a
	}
	newRight := pqMerge(a.Right, b)
	// leftist property: the left subtree's rank is at least the right's
	if pqRank(a.Left) >= pqRank(newRight) {
		return &PQHeap{Prio: a.Prio, Val: a.Val, Left: a.Left, Right: newRight, Rank: pqRank(newRight) + 1}
	}
	return &PQHeap{Prio: a.Prio, Val: a.Val, Left: newRight, Right: a.Left, Rank: pqRank(a.Left) + 1}
}

// Insert returns a new heap with (prio, val) added.
func (h *PQHeap) Insert(prio int64, val Node) *PQHeap {
	return pqMerge(h, &PQHeap{Prio: prio, Val: val, Rank: 1})
}

// DeleteMin returns the heap without its minimum element (nil-safe).
func (h *PQHeap) DeleteMin() *PQHeap {
	if h == nil {
		return nil
	}
	return pqMerge(h.Left, h.Right)
}
