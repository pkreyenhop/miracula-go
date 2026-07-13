package ast

// MapKey is a Miracula map key: either an int64 or a string (the type
// checker unifies key types, so any one map uses a single key kind).
// Integer keys stay integers — no per-operation string formatting.
type MapKey struct {
	I   int64
	S   string
	Str bool
}

// Less orders keys: integers before strings, then by value.
func (a MapKey) Less(b MapKey) bool {
	if a.Str != b.Str {
		return !a.Str
	}
	if a.Str {
		return a.S < b.S
	}
	return a.I < b.I
}

// MapTree is an immutable AVL tree: Insert returns a new tree sharing all
// untouched nodes with the original, so h_insert is O(log n) time and
// space instead of a full-map copy.
type MapTree struct {
	Key   MapKey
	Val   Node
	Left  *MapTree
	Right *MapTree
	H     int8
}

func treeHeight(t *MapTree) int8 {
	if t == nil {
		return 0
	}
	return t.H
}

func mkTree(k MapKey, v Node, l, r *MapTree) *MapTree {
	h := treeHeight(l)
	if rh := treeHeight(r); rh > h {
		h = rh
	}
	return &MapTree{Key: k, Val: v, Left: l, Right: r, H: h + 1}
}

// balanceTree builds a node for (k, v, l, r), rotating if the AVL
// invariant is violated by at most one recent insertion.
func balanceTree(k MapKey, v Node, l, r *MapTree) *MapTree {
	bf := treeHeight(l) - treeHeight(r)
	if bf > 1 {
		if treeHeight(l.Left) >= treeHeight(l.Right) {
			return mkTree(l.Key, l.Val, l.Left, mkTree(k, v, l.Right, r))
		}
		lr := l.Right
		return mkTree(lr.Key, lr.Val, mkTree(l.Key, l.Val, l.Left, lr.Left), mkTree(k, v, lr.Right, r))
	}
	if bf < -1 {
		if treeHeight(r.Right) >= treeHeight(r.Left) {
			return mkTree(r.Key, r.Val, mkTree(k, v, l, r.Left), r.Right)
		}
		rl := r.Left
		return mkTree(rl.Key, rl.Val, mkTree(k, v, l, rl.Left), mkTree(r.Key, r.Val, rl.Right, r.Right))
	}
	return mkTree(k, v, l, r)
}

// Insert returns a tree with k bound to v, replacing any existing binding.
func (t *MapTree) Insert(k MapKey, v Node) *MapTree {
	if t == nil {
		return &MapTree{Key: k, Val: v, H: 1}
	}
	if k.Less(t.Key) {
		return balanceTree(t.Key, t.Val, t.Left.Insert(k, v), t.Right)
	}
	if t.Key.Less(k) {
		return balanceTree(t.Key, t.Val, t.Left, t.Right.Insert(k, v))
	}
	return &MapTree{Key: k, Val: v, Left: t.Left, Right: t.Right, H: t.H}
}

// Lookup finds the value bound to k, if any. Allocation-free.
func (t *MapTree) Lookup(k MapKey) (Node, bool) {
	for t != nil {
		if k.Less(t.Key) {
			t = t.Left
		} else if t.Key.Less(k) {
			t = t.Right
		} else {
			return t.Val, true
		}
	}
	return nil, false
}
