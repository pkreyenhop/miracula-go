package ast

import (
	"sync"
	"unsafe"
)

// ==========================================================================
// 1. AST & VALUE NODES DEFINITION
// ==========================================================================

type Node interface {
	isNode()
}

type IntNode struct{ Val int64 }
type BoolNode struct{ Val bool }
type CharNode struct{ Val rune }
type NilNode struct{}
type ConsNode struct{ Head, Tail Node }
type TupleNode struct{ Elems []Node }
type VarNode struct{ Name string }

// LocalVarNode is a VarNode resolved to lexical coordinates: the binding
// lives Depth single-binding environment frames above the use site, so the
// evaluator reaches it with Depth pointer hops and no name comparison.
// Name is kept for error messages only.
type LocalVarNode struct {
	Depth int
	Name  string
}

// GlobalVarNode is a VarNode resolved to a script-level definition: the
// evaluator reads it straight from the Globals map without walking the
// local chain first.
type GlobalVarNode struct{ Name string }
type LamNode struct {
	Var  string
	Body Node
}
type ClosureNode struct {
	Var  string
	Body Node
	Env  *Env
}
type LetNode struct {
	Bindings []Binding
	Body     Node
}
type ProjNode struct {
	Index int
	Tuple Node
}
type AppNode struct {
	Left  Node
	Right Node
}
type MatchErrorNode struct{}

type MapNode struct {
	Tree *MapTree
}

type SetNode struct {
	Tree *MapTree
}

type HLookupPartialNode struct {
	Tree *MapTree
}

type HInsertPartialNode1 struct {
	Tree *MapTree
}

type HInsertPartialNode2 struct {
	Tree *MapTree
	Key  MapKey
}

type MemberPartialNode struct {
	Tree *MapTree
}

type SInsertPartialNode struct {
	Tree *MapTree
}

type SeqPartialNode struct{}

type SplitPartialNode struct {
	Delims string
}

type ListGetPartialNode struct {
	List []int64
}

type ListSetPartialNode1 struct {
	List []int64
}

type ListSetPartialNode2 struct {
	List  []int64
	Index int64
}

type MemoizeNode struct {
	Func     Node
	Cache    map[string]Node
	IntCache map[int64]Node
}

type SortByPartialNode struct {
	Cmp Node
}

// BitopPartialNode is a bitwise builtin (xor/band/bor/shl/shr) applied to its
// first integer argument, awaiting the second.
type BitopPartialNode struct {
	Op string
	A  int64
}

// MemoFixNode is the result of `memofix f`: a memoized fixpoint. Applying it to
// an argument x evaluates `f self x` (so f's first parameter is the recursive
// call) and caches the result by x, exactly like MemoizeNode. The caches are
// Go maps, shared across copies of the node, so recursive calls hit them.
type MemoFixNode struct {
	Func     Node
	Cache    map[string]Node
	IntCache map[int64]Node
}

// PQNode is a priority-queue value wrapping a persistent min-heap.
type PQNode struct {
	Heap *PQHeap
}

// PQPushPartial1/2 are pq_push (3-arg) applied to one/two of its arguments.
type PQPushPartial1 struct {
	Heap *PQHeap
}
type PQPushPartial2 struct {
	Heap *PQHeap
	Prio int64
}

type HLookupDefPartialNode1 struct {
	Tree *MapTree
}

type HLookupDefPartialNode2 struct {
	Tree *MapTree
	Key  MapKey
}

type VecNode struct {
	Elems []Node
}

type VecGetPartialNode struct {
	Elems []Node
}

type VecSetPartialNode1 struct {
	Elems []Node
}

type VecSetPartialNode2 struct {
	Elems []Node
	Index int64
}

type ThunkState int

const (
	Unevaluated ThunkState = iota
	Evaluating
	Evaluated
)

type ThunkCell struct {
	State ThunkState
	Expr  Node
	Env   *Env
	Val   Node
}

type ThunkNode struct {
	Cell *ThunkCell
}

type IfZeroNode struct{ Cond, Then, Else Node }
type IfNilNode struct{ Cond, Then, Else Node }
type IfNode struct{ Cond, Then, Else Node }
type AppendNode struct{ Left, Right Node }
type DiffNode struct{ Left, Right Node }
type RangeNode struct{ Start, End Node }

// RangeFromNode is an unbounded range [start..]: an infinite lazy list of
// consecutive integers.
type RangeFromNode struct{ Start Node }

type ZFNode struct {
	Body  Node
	Quals []Qualifier
}

type ZFGeneratorNode struct {
	Pat   Pat
	Rest  []Qualifier
	Src   Node
	Body  Node
	ZFEnv *Env
}

type AddNode struct{ Left, Right Node }
type SubNode struct{ Left, Right Node }
type MulNode struct{ Left, Right Node }
type DivNode struct{ Left, Right Node }
type ModNode struct{ Left, Right Node }
type EqNode struct{ Left, Right Node }
type NeNode struct{ Left, Right Node }
type LtNode struct{ Left, Right Node }
type GtNode struct{ Left, Right Node }
type LeNode struct{ Left, Right Node }
type GeNode struct{ Left, Right Node }

func (IntNode) isNode()                {}
func (BoolNode) isNode()               {}
func (CharNode) isNode()               {}
func (NilNode) isNode()                {}
func (ConsNode) isNode()               {}
func (TupleNode) isNode()              {}
func (VarNode) isNode()                {}
func (LocalVarNode) isNode()           {}
func (GlobalVarNode) isNode()          {}
func (LamNode) isNode()                {}
func (ClosureNode) isNode()            {}
func (LetNode) isNode()                {}
func (ProjNode) isNode()               {}
func (AppNode) isNode()                {}
func (MatchErrorNode) isNode()         {}
func (ThunkNode) isNode()              {}
func (IfZeroNode) isNode()             {}
func (IfNilNode) isNode()              {}
func (IfNode) isNode()                 {}
func (AppendNode) isNode()             {}
func (DiffNode) isNode()               {}
func (ZFNode) isNode()                 {}
func (ZFGeneratorNode) isNode()        {}
func (RangeNode) isNode()              {}
func (RangeFromNode) isNode()          {}
func (AddNode) isNode()                {}
func (SubNode) isNode()                {}
func (MulNode) isNode()                {}
func (DivNode) isNode()                {}
func (ModNode) isNode()                {}
func (EqNode) isNode()                 {}
func (NeNode) isNode()                 {}
func (LtNode) isNode()                 {}
func (GtNode) isNode()                 {}
func (LeNode) isNode()                 {}
func (GeNode) isNode()                 {}
func (MapNode) isNode()                {}
func (SetNode) isNode()                {}
func (HLookupPartialNode) isNode()     {}
func (HInsertPartialNode1) isNode()    {}
func (HInsertPartialNode2) isNode()    {}
func (MemberPartialNode) isNode()      {}
func (SInsertPartialNode) isNode()     {}
func (SeqPartialNode) isNode()         {}
func (SplitPartialNode) isNode()       {}
func (ListGetPartialNode) isNode()     {}
func (ListSetPartialNode1) isNode()    {}
func (ListSetPartialNode2) isNode()    {}
func (MemoizeNode) isNode()            {}
func (SortByPartialNode) isNode()      {}
func (BitopPartialNode) isNode()       {}
func (MemoFixNode) isNode()            {}
func (PQNode) isNode()                 {}
func (PQPushPartial1) isNode()         {}
func (PQPushPartial2) isNode()         {}
func (HLookupDefPartialNode1) isNode() {}
func (HLookupDefPartialNode2) isNode() {}
func (VecNode) isNode()                {}
func (VecGetPartialNode) isNode()      {}
func (VecSetPartialNode1) isNode()     {}
func (VecSetPartialNode2) isNode()     {}

type Binding struct {
	Name string
	Expr Node
}

// ==========================================================================
// 2. PATTERNS & QUALIFIERS
// ==========================================================================

type Pat interface {
	isPat()
}

type PatInt struct{ Val int64 }
type PatBool struct{ Val bool }
type PatChar struct{ Val rune }
type PatVar struct{ Name string }
type PatNil struct{}
type PatCons struct{ Head, Tail Pat }
type PatTuple struct{ Elems []Pat }

func (PatInt) isPat()   {}
func (PatBool) isPat()  {}
func (PatChar) isPat()  {}
func (PatVar) isPat()   {}
func (PatNil) isPat()   {}
func (PatCons) isPat()  {}
func (PatTuple) isPat() {}

type Qualifier interface {
	isQualifier()
}

type GeneratorQual struct {
	Pat Pat
	Src Node
}

type FilterQual struct {
	Cond Node
}

func (GeneratorQual) isQualifier() {}
func (FilterQual) isQualifier()    {}

// ==========================================================================
// 3. ENVIRONMENT DEFINITION
// ==========================================================================

type Env struct {
	Parent  *Env
	Name    string
	Val     Node
	Globals map[string]Node
	// Root is the base frame of the chain (no local bindings, Globals set).
	// Evaluating a global definition switches to it so caller locals are
	// not captured, without allocating a fresh environment per reference.
	Root *Env
}

func (e *Env) Lookup(x string) (Node, bool) {
	for curr := e; curr != nil; curr = curr.Parent {
		if curr.Name == x {
			return curr.Val, true
		}
	}
	if e != nil && e.Globals != nil {
		if val, ok := e.Globals[x]; ok {
			return val, true
		}
	}
	return nil, false
}

func (e *Env) Extend(x string, val Node) *Env {
	var globals map[string]Node
	var root *Env
	if e != nil {
		globals = e.Globals
		root = e.Root
	}
	return &Env{Parent: e, Name: x, Val: val, Globals: globals, Root: root}
}

func (e *Env) ExtendGlobal(x string, val Node) *Env {
	if e.Globals == nil {
		e.Globals = make(map[string]Node)
	}
	e.Globals[x] = val
	return e
}

func (e *Env) GetNames() []string {
	var names []string
	seen := make(map[string]bool)
	for curr := e; curr != nil; curr = curr.Parent {
		if curr.Name != "" && !seen[curr.Name] {
			seen[curr.Name] = true
			names = append(names, curr.Name)
		}
	}
	if e != nil && e.Globals != nil {
		for k := range e.Globals {
			if !seen[k] {
				seen[k] = true
				names = append(names, k)
			}
		}
	}
	return names
}

func NewEnv() *Env {
	globals := make(map[string]Node)
	globals["True"] = BoolNode{Val: true}
	globals["False"] = BoolNode{Val: false}
	e := &Env{Globals: globals}
	e.Root = e
	return e
}

// ==========================================================================
// 4. EXCEPTIONS / ERRORS
// ==========================================================================

type RuntimeError struct {
	Msg string
}

func (e RuntimeError) Error() string { return e.Msg }

type BlackholeError struct {
	Msg string
}

func (e BlackholeError) Error() string { return e.Msg }

type Position struct {
	Filename string
	Line     int
	Col      int
}

type interfaceHeader struct {
	Type unsafe.Pointer
	Data unsafe.Pointer
}

func GetNodeKey(node Node) unsafe.Pointer {
	return (*interfaceHeader)(unsafe.Pointer(&node)).Data
}

var NodePositions sync.Map // map[unsafe.Pointer]Position
