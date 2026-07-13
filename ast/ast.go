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

type IntNode struct{ Val int }
type BoolNode struct{ Val bool }
type CharNode struct{ Val rune }
type NilNode struct{}
type ConsNode struct{ Head, Tail Node }
type TupleNode struct{ Elems []Node }
type VarNode struct{ Name string }
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

type ZFNode struct {
	Body Node
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

func (IntNode) isNode()         {}
func (BoolNode) isNode()        {}
func (CharNode) isNode()        {}
func (NilNode) isNode()         {}
func (ConsNode) isNode()        {}
func (TupleNode) isNode()       {}
func (VarNode) isNode()         {}
func (LamNode) isNode()         {}
func (ClosureNode) isNode()     {}
func (LetNode) isNode()         {}
func (ProjNode) isNode()        {}
func (AppNode) isNode()         {}
func (MatchErrorNode) isNode()  {}
func (ThunkNode) isNode()       {}
func (IfZeroNode) isNode()      {}
func (IfNilNode) isNode()       {}
func (IfNode) isNode()          {}
func (AppendNode) isNode()      {}
func (DiffNode) isNode()        {}
func (ZFNode) isNode()          {}
func (ZFGeneratorNode) isNode() {}
func (RangeNode) isNode()       {}
func (AddNode) isNode()         {}
func (SubNode) isNode()         {}
func (MulNode) isNode()         {}
func (DivNode) isNode()         {}
func (ModNode) isNode()         {}
func (EqNode) isNode()          {}
func (NeNode) isNode()          {}
func (LtNode) isNode()          {}
func (GtNode) isNode()          {}
func (LeNode) isNode()          {}
func (GeNode) isNode()          {}


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

type PatInt struct{ Val int }
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
	Parent *Env
	Name   string
	Val    Node
}

func (e *Env) Lookup(x string) (Node, bool) {
	for curr := e; curr != nil; curr = curr.Parent {
		if curr.Name == x {
			return curr.Val, true
		}
	}
	return nil, false
}

func (e *Env) Extend(x string, val Node) *Env {
	return &Env{Parent: e, Name: x, Val: val}
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
	return names
}

func NewEnv() *Env {
	env := &Env{}
	env = env.Extend("True", BoolNode{Val: true})
	env = env.Extend("False", BoolNode{Val: false})
	return env
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
