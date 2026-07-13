package eval

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"unicode"

	"pkreyenhop.com/miracula-go/ast"
)

var interrupted int32

func SetInterrupted(val bool) {
	if val {
		atomic.StoreInt32(&interrupted, 1)
	} else {
		atomic.StoreInt32(&interrupted, 0)
	}
}

func IsInterrupted() bool {
	return atomic.LoadInt32(&interrupted) == 1
}

type InterruptedException struct{}

func smlDiv(a, b int64) int64 {
	q := a / b
	r := a % b
	if (r > 0 && b < 0) || (r < 0 && b > 0) {
		q--
	}
	return q
}

func smlMod(a, b int64) int64 {
	r := a % b
	if (r > 0 && b < 0) || (r < 0 && b > 0) {
		r += b
	}
	return r
}

func needsThunkCons(n ast.Node) bool {
	switch n.(type) {
	case ast.IntNode, ast.BoolNode, ast.CharNode, ast.NilNode, ast.ThunkNode, ast.ClosureNode, ast.MatchErrorNode:
		return false
	}
	return true
}

// bindArg prepares a function argument for binding into the callee
// environment. Values already in WHNF pass through unchanged, and an
// argument that is just a variable reference passes its existing local
// binding through instead of allocating a fresh indirection thunk —
// repeated re-thunking otherwise builds arbitrarily deep thunk chains
// (e.g. an accumulator threaded through a long tail-recursive loop).
func bindArg(env *ast.Env, arg ast.Node) ast.Node {
	switch r := arg.(type) {
	case ast.IntNode, ast.BoolNode, ast.CharNode, ast.NilNode, ast.ClosureNode, ast.ThunkNode, ast.MatchErrorNode:
		return r
	case ast.LamNode:
		return ast.ClosureNode{Var: r.Var, Body: r.Body, Env: env}
	case ast.VarNode:
		for curr := env; curr != nil; curr = curr.Parent {
			if curr.Name == r.Name {
				return curr.Val
			}
		}
	case ast.LocalVarNode:
		curr := env
		for d := r.Depth; d > 0 && curr != nil; d-- {
			curr = curr.Parent
		}
		if curr != nil {
			return curr.Val
		}
	}
	return ast.ThunkNode{Cell: &ast.ThunkCell{State: ast.Unevaluated, Expr: arg, Env: env}}
}

func needsThunkTuple(n ast.Node) bool {
	switch n.(type) {
	case ast.IntNode, ast.BoolNode, ast.CharNode, ast.NilNode, ast.ThunkNode, ast.ClosureNode, ast.MatchErrorNode:
		return false
	}
	return true
}

type MatchBinding struct {
	Name string
	Val  ast.Node
}

func mergeBindings(m1, m2 []MatchBinding) []MatchBinding {
	res := append([]MatchBinding(nil), m1...)
	for _, b2 := range m2 {
		found := false
		for i, b1 := range res {
			if b1.Name == b2.Name {
				res[i].Val = b2.Val
				found = true
				break
			}
		}
		if !found {
			res = append(res, b2)
		}
	}
	return res
}

func matchPattern(env *ast.Env, pat ast.Pat, node ast.Node) ([]MatchBinding, bool) {
	v := Whnf(env, node)
	switch p := pat.(type) {
	case ast.PatInt:
		if i, ok := v.(ast.IntNode); ok && p.Val == i.Val {
			return nil, true
		}
		return nil, false
	case ast.PatBool:
		if b, ok := v.(ast.BoolNode); ok && p.Val == b.Val {
			return nil, true
		}
		return nil, false
	case ast.PatChar:
		if c, ok := v.(ast.CharNode); ok && p.Val == c.Val {
			return nil, true
		}
		return nil, false
	case ast.PatVar:
		if p.Name == "_" {
			return nil, true
		}
		return []MatchBinding{{Name: p.Name, Val: v}}, true
	case ast.PatNil:
		if _, ok := v.(ast.NilNode); ok {
			return nil, true
		}
		return nil, false
	case ast.PatCons:
		if c, ok := v.(ast.ConsNode); ok {
			m1, ok1 := matchPattern(env, p.Head, c.Head)
			if !ok1 {
				return nil, false
			}
			m2, ok2 := matchPattern(env, p.Tail, c.Tail)
			if !ok2 {
				return nil, false
			}
			return mergeBindings(m1, m2), true
		}
		return nil, false
	case ast.PatTuple:
		if t, ok := v.(ast.TupleNode); ok {
			if len(p.Elems) != len(t.Elems) {
				return nil, false
			}
			var acc []MatchBinding
			for i := range p.Elems {
				m, ok := matchPattern(env, p.Elems[i], t.Elems[i])
				if !ok {
					return nil, false
				}
				acc = mergeBindings(acc, m)
			}
			return acc, true
		}
		return nil, false
	}
	return nil, false
}

func getStringValue(env *ast.Env, node ast.Node) string {
	var sb strings.Builder
	curr := node
	for {
		switch l := Whnf(env, curr).(type) {
		case ast.NilNode:
			return sb.String()
		case ast.ConsNode:
			hVal := Whnf(env, l.Head)
			if c, ok := hVal.(ast.CharNode); ok {
				sb.WriteRune(c.Val)
				curr = l.Tail
				continue
			}
			panic(ast.RuntimeError{Msg: "Expected char in string"})
		default:
			panic(ast.RuntimeError{Msg: "Expected string"})
		}
	}
}

func getMapKey(env *ast.Env, node ast.Node) string {
	v := Whnf(env, node)
	if i, ok := v.(ast.IntNode); ok {
		return strconv.FormatInt(i.Val, 10)
	}
	return getStringValue(env, v)
}

func MakeStringNode(s string) ast.Node {
	runes := []rune(s)
	var listNode ast.Node = ast.NilNode{}
	for i := len(runes) - 1; i >= 0; i-- {
		listNode = ast.ConsNode{Head: ast.CharNode{Val: runes[i]}, Tail: listNode}
	}
	return listNode
}

func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	fields := strings.Split(s, "\n")
	if len(fields) > 0 && fields[len(fields)-1] == "" {
		fields = fields[:len(fields)-1]
	}
	return fields
}

func removeOne(env *ast.Env, x ast.Node, listNode ast.Node) ast.Node {
	l := Whnf(env, listNode)
	switch xs := l.(type) {
	case ast.NilNode:
		return ast.NilNode{}
	case ast.ConsNode:
		eqH := Whnf(env, ast.EqNode{Left: x, Right: xs.Head})
		if isTrueNode(eqH) {
			return xs.Tail
		}
		tEval := removeOne(env, x, xs.Tail)
		return ast.ConsNode{Head: xs.Head, Tail: tEval}
	default:
		panic(ast.RuntimeError{Msg: "-- expects lists"})
	}
}

func diff(env *ast.Env, xs, ys ast.Node) ast.Node {
	currY := ys
	currX := xs
	for {
		yVal := Whnf(env, currY)
		switch y := yVal.(type) {
		case ast.NilNode:
			return currX
		case ast.ConsNode:
			yEval := Whnf(env, y.Head)
			currX = removeOne(env, yEval, currX)
			currY = y.Tail
		default:
			panic(ast.RuntimeError{Msg: "-- expects lists"})
		}
	}
}

func evalZF(env *ast.Env, bodyExpr ast.Node, quals []ast.Qualifier) ast.Node {
	if len(quals) == 0 {
		h := bodyExpr
		if needsThunkCons(bodyExpr) {
			h = ast.ThunkNode{Cell: &ast.ThunkCell{State: ast.Unevaluated, Expr: bodyExpr, Env: env}}
		}
		return ast.ConsNode{Head: h, Tail: ast.NilNode{}}
	}
	q := quals[0]
	switch qual := q.(type) {
	case ast.FilterQual:
		cond := ast.ThunkNode{Cell: &ast.ThunkCell{State: ast.Unevaluated, Expr: qual.Cond, Env: env}}
		return ast.IfNode{Cond: cond, Then: evalZF(env, bodyExpr, quals[1:]), Else: ast.NilNode{}}
	case ast.GeneratorQual:
		return ast.ZFGeneratorNode{
			Pat:   qual.Pat,
			Rest:  quals[1:],
			Src:   qual.Src,
			Body:  bodyExpr,
			ZFEnv: env,
		}
	}
	panic("Unknown qualifier type")
}

func isTrueNode(n ast.Node) bool {
	if b, ok := n.(ast.BoolNode); ok {
		return b.Val
	}
	if i, ok := n.(ast.IntNode); ok {
		return i.Val != 0
	}
	return false
}

func eq(env *ast.Env, v1, v2 ast.Node) bool {
	for {
		switch x1 := v1.(type) {
		case ast.IntNode:
			if x2, ok := v2.(ast.IntNode); ok {
				return x1.Val == x2.Val
			}
		case ast.BoolNode:
			if x2, ok := v2.(ast.BoolNode); ok {
				return x1.Val == x2.Val
			}
		case ast.CharNode:
			if x2, ok := v2.(ast.CharNode); ok {
				return x1.Val == x2.Val
			}
		case ast.NilNode:
			switch v2.(type) {
			case ast.NilNode:
				return true
			case ast.ConsNode:
				return false
			}
		case ast.ConsNode:
			switch x2 := v2.(type) {
			case ast.NilNode:
				return false
			case ast.ConsNode:
				eqH := Whnf(env, ast.EqNode{Left: x1.Head, Right: x2.Head})
				if !isTrueNode(eqH) {
					return false
				}
				// walk the tails iteratively so comparing long lists
				// does not consume one Go stack frame per element
				v1 = Whnf(env, x1.Tail)
				v2 = Whnf(env, x2.Tail)
				continue
			}
		case ast.TupleNode:
			if x2, ok := v2.(ast.TupleNode); ok {
				if len(x1.Elems) != len(x2.Elems) {
					return false
				}
				for i := range x1.Elems {
					eqE := Whnf(env, ast.EqNode{Left: x1.Elems[i], Right: x2.Elems[i]})
					if !isTrueNode(eqE) {
						return false
					}
				}
				return true
			}
		}
		panic(ast.RuntimeError{Msg: fmt.Sprintf("Equality expects integers, booleans, characters, lists or tuples, got: %s and %s", PrintNode(env, v1), PrintNode(env, v2))})
	}
}

func Whnf(env *ast.Env, n ast.Node) ast.Node {
	if IsInterrupted() {
		panic(InterruptedException{})
	}
	// small inline buffer: most evaluations record only a few pending
	// thunks, so the common case never touches the heap
	var pbuf [8]*ast.ThunkCell
	pending := pbuf[:0]
	v := whnfCore(env, n, &pending)
	for _, cell := range pending {
		cell.Val = v
		cell.State = ast.Evaluated
	}
	return v
}

// whnfCore reduces n to weak head normal form on an explicit control
// stack. An unevaluated thunk reached in tail position is recorded in
// pending (Whnf writes the final value into cells left over at return);
// strict operand positions push a continuation frame carrying a pending
// watermark, so cells recorded during an operand's evaluation receive
// exactly that operand's value. Neither chains of dependent thunks nor
// strict-operand nesting consume Go stack frames; nested Whnf calls remain
// only in bounded positions (builtin internals, range bounds, list diff,
// structural equality of heads, pattern matching).
// continuation kinds for whnfCore's explicit control stack: strict operand
// positions push a continuation instead of making a nested recursive call,
// so evaluation depth is bounded by the heap, not by the Go stack.
const (
	kIfCond = iota
	kIfZeroCond
	kIfNilCond
	kBinopLeft
	kBinopRight
	kAppFun
	kProjTuple
	kAppendLeft
)

// cont is one frame of the explicit control stack. mark records the length
// of the pending-thunk stack when the frame was pushed: thunk cells recorded
// while evaluating this frame's operand sit above mark and receive exactly
// that operand's value when it arrives. The node/val fields are stage
// dependent: kBinopLeft holds the operator node and the unevaluated right
// operand in val; kBinopRight holds the operator node and the evaluated left
// operand in val; kIf* hold the conditional node; kAppFun holds the argument
// expression; kAppendLeft holds the right list expression.
type cont struct {
	kind int
	mark int
	idx  int
	node ast.Node
	val  ast.Node
	env  *ast.Env
}

// evalBinop applies a binary operator node to two evaluated operands.
func evalBinop(env *ast.Env, op ast.Node, v1, v2 ast.Node) ast.Node {
	switch op.(type) {
	case ast.AddNode:
		i1, ok1 := v1.(ast.IntNode)
		i2, ok2 := v2.(ast.IntNode)
		if !ok1 || !ok2 {
			panic(ast.RuntimeError{Msg: "Addition expects integers"})
		}
		return ast.IntNode{Val: i1.Val + i2.Val}
	case ast.SubNode:
		i1, ok1 := v1.(ast.IntNode)
		i2, ok2 := v2.(ast.IntNode)
		if !ok1 || !ok2 {
			panic(ast.RuntimeError{Msg: "Subtraction expects integers"})
		}
		return ast.IntNode{Val: i1.Val - i2.Val}
	case ast.MulNode:
		i1, ok1 := v1.(ast.IntNode)
		i2, ok2 := v2.(ast.IntNode)
		if !ok1 || !ok2 {
			panic(ast.RuntimeError{Msg: "Multiplication expects integers"})
		}
		return ast.IntNode{Val: i1.Val * i2.Val}
	case ast.DivNode:
		i1, ok1 := v1.(ast.IntNode)
		i2, ok2 := v2.(ast.IntNode)
		if !ok1 || !ok2 {
			panic(ast.RuntimeError{Msg: "Division expects integers"})
		}
		if i2.Val == 0 {
			panic(ast.RuntimeError{Msg: "Division by zero"})
		}
		return ast.IntNode{Val: smlDiv(i1.Val, i2.Val)}
	case ast.ModNode:
		i1, ok1 := v1.(ast.IntNode)
		i2, ok2 := v2.(ast.IntNode)
		if !ok1 || !ok2 {
			panic(ast.RuntimeError{Msg: "Modulo expects integers"})
		}
		if i2.Val == 0 {
			panic(ast.RuntimeError{Msg: "Division by zero"})
		}
		return ast.IntNode{Val: smlMod(i1.Val, i2.Val)}
	case ast.EqNode:
		return ast.BoolNode{Val: eq(env, v1, v2)}
	case ast.NeNode:
		return ast.BoolNode{Val: !eq(env, v1, v2)}
	case ast.LtNode:
		i1, ok1 := v1.(ast.IntNode)
		i2, ok2 := v2.(ast.IntNode)
		if !ok1 || !ok2 {
			panic(ast.RuntimeError{Msg: "Less-than expects integers"})
		}
		return ast.BoolNode{Val: i1.Val < i2.Val}
	case ast.GtNode:
		i1, ok1 := v1.(ast.IntNode)
		i2, ok2 := v2.(ast.IntNode)
		if !ok1 || !ok2 {
			panic(ast.RuntimeError{Msg: "Greater-than expects integers"})
		}
		return ast.BoolNode{Val: i1.Val > i2.Val}
	case ast.LeNode:
		i1, ok1 := v1.(ast.IntNode)
		i2, ok2 := v2.(ast.IntNode)
		if !ok1 || !ok2 {
			panic(ast.RuntimeError{Msg: "Less-than-or-equal expects integers"})
		}
		return ast.BoolNode{Val: i1.Val <= i2.Val}
	case ast.GeNode:
		i1, ok1 := v1.(ast.IntNode)
		i2, ok2 := v2.(ast.IntNode)
		if !ok1 || !ok2 {
			panic(ast.RuntimeError{Msg: "Greater-than-or-equal expects integers"})
		}
		return ast.BoolNode{Val: i1.Val >= i2.Val}
	}
	panic(fmt.Sprintf("Internal error: unknown binary operator: %T", op))
}

func whnfCore(env *ast.Env, n ast.Node, pending *[]*ast.ThunkCell) ast.Node {
	// small inline buffer: shallow expressions (the overwhelmingly common
	// case for nested Whnf calls) never allocate a control stack
	var cbuf [16]cont
	conts := cbuf[:0]
	steps := 0
machine:
	for {
		steps++
		if steps&4095 == 0 && IsInterrupted() {
			panic(InterruptedException{})
		}
		var v ast.Node
	dispatch:
		switch node := n.(type) {
		case ast.IntNode:
			v = node
		case ast.BoolNode:
			v = node
		case ast.CharNode:
			v = node
		case ast.NilNode:
			v = node
		case ast.MapNode:
			v = node
		case ast.SetNode:
			v = node
		case ast.HLookupPartialNode:
			v = node
		case ast.HInsertPartialNode1:
			v = node
		case ast.HInsertPartialNode2:
			v = node
		case ast.MemberPartialNode:
			v = node
		case ast.SeqPartialNode:
			v = node
		case ast.SplitPartialNode:
			v = node
		case ast.ListGetPartialNode:
			v = node
		case ast.ListSetPartialNode1:
			v = node
		case ast.ListSetPartialNode2:
			v = node
		case ast.MemoizeNode:
			v = node
		case ast.SortByPartialNode:
			v = node
		case ast.HLookupDefPartialNode1:
			v = node
		case ast.HLookupDefPartialNode2:
			v = node
		case ast.LamNode:
			v = ast.ClosureNode{Var: node.Var, Body: node.Body, Env: env}
		case ast.ClosureNode:
			v = node
		case ast.LetNode:
			envPrime := env
			cells := make([]*ast.ThunkCell, len(node.Bindings))
			for i, b := range node.Bindings {
				cells[i] = &ast.ThunkCell{State: ast.Unevaluated, Expr: b.Expr}
				envPrime = envPrime.Extend(b.Name, ast.ThunkNode{Cell: cells[i]})
			}
			for _, cell := range cells {
				cell.Env = envPrime
			}
			env = envPrime
			n = node.Body
			continue
		case ast.ConsNode:
			h := node.Head
			t := node.Tail
			if needsThunkCons(h) {
				h = ast.ThunkNode{Cell: &ast.ThunkCell{State: ast.Unevaluated, Expr: h, Env: env}}
			}
			if needsThunkCons(t) {
				t = ast.ThunkNode{Cell: &ast.ThunkCell{State: ast.Unevaluated, Expr: t, Env: env}}
			}
			v = ast.ConsNode{Head: h, Tail: t}
		case ast.TupleNode:
			elmsPrime := make([]ast.Node, len(node.Elems))
			for i, e := range node.Elems {
				if needsThunkTuple(e) {
					elmsPrime[i] = ast.ThunkNode{Cell: &ast.ThunkCell{State: ast.Unevaluated, Expr: e, Env: env}}
				} else {
					elmsPrime[i] = e
				}
			}
			v = ast.TupleNode{Elems: elmsPrime}
		case ast.LocalVarNode:
			curr := env
			for d := node.Depth; d > 0; d-- {
				curr = curr.Parent
				if curr == nil {
					panic(ast.RuntimeError{Msg: "Internal error: bad lexical depth for: " + node.Name})
				}
			}
			val := curr.Val
			if th, ok := val.(ast.ThunkNode); ok {
				cell := th.Cell
				switch cell.State {
				case ast.Evaluated:
					switch cv := cell.Val.(type) {
					case ast.IntNode, ast.BoolNode, ast.CharNode, ast.NilNode, ast.ClosureNode, ast.MatchErrorNode:
						v = cv
						break dispatch
					default:
						n = cv
						continue
					}
				case ast.Evaluating:
					panic(ast.BlackholeError{Msg: "Infinite loop on identifier: " + node.Name})
				case ast.Unevaluated:
					cell.State = ast.Evaluating
					*pending = append(*pending, cell)
					n = cell.Expr
					env = cell.Env
					continue
				}
			}
			n = val
			continue
		case ast.GlobalVarNode:
			gv, gok := env.Globals[node.Name]
			if !gok {
				panic(ast.RuntimeError{Msg: "Unbound variable: " + node.Name})
			}
			// globals are closed terms over the global scope: evaluate them
			// in the chain's root frame so caller locals are not captured
			if env.Root != nil {
				env = env.Root
			} else {
				env = &ast.Env{Globals: env.Globals}
			}
			n = gv
			continue
		case ast.VarNode:
			name := node.Name
			switch name {
			case "hd", "tl", "show", "read", "lines", "numval", "length", "reverse", "seq", "h_lookup", "h_insert", "member", "split", "parse_ints", "list_get", "list_set", "memoize", "sort_by", "sort_ints", "sort_edges", "sort_pts", "h_lookup_def":
				v = node
			case "empty_map":
				v = ast.MapNode{Map: make(map[string]ast.Node)}
			case "empty_set":
				v = ast.SetNode{Set: make(map[string]bool)}
			}
			if v != nil {
				break
			}

			var val ast.Node
			var ok bool
			for curr := env; curr != nil; curr = curr.Parent {
				if curr.Name == name {
					val = curr.Val
					ok = true
					break
				}
			}
			if !ok && env.Globals != nil {
				if gv, gok := env.Globals[name]; gok {
					// Globals are closed terms over the global scope: evaluate
					// them in a globals-only environment so caller locals are
					// not captured (static scoping; also keeps the environment
					// chain from growing on every call).
					if env.Root != nil {
						env = env.Root
					} else {
						env = &ast.Env{Globals: env.Globals}
					}
					val = gv
					ok = true
				}
			}
			if !ok {
				panic(ast.RuntimeError{Msg: "Unbound variable: " + name})
			}
			if th, ok := val.(ast.ThunkNode); ok {
				cell := th.Cell
				switch cell.State {
				case ast.Evaluated:
					switch cv := cell.Val.(type) {
					case ast.IntNode, ast.BoolNode, ast.CharNode, ast.NilNode, ast.ClosureNode, ast.MatchErrorNode:
						v = cv
						break dispatch
					default:
						n = cv
						continue
					}
				case ast.Evaluating:
					panic(ast.BlackholeError{Msg: "Infinite loop on identifier: " + name})
				case ast.Unevaluated:
					cell.State = ast.Evaluating
					*pending = append(*pending, cell)
					n = cell.Expr
					env = cell.Env
					continue
				}
			}
			n = val
			continue
		case ast.ThunkNode:
			cell := node.Cell
			switch cell.State {
			case ast.Evaluated:
				switch cv := cell.Val.(type) {
				case ast.IntNode, ast.BoolNode, ast.CharNode, ast.NilNode, ast.ClosureNode, ast.MatchErrorNode:
					v = cv
					break dispatch
				default:
					n = cv
					continue
				}
			case ast.Evaluating:
				panic(ast.BlackholeError{Msg: "Infinite loop inside generic thunk node"})
			case ast.Unevaluated:
				cell.State = ast.Evaluating
				*pending = append(*pending, cell)
				n = cell.Expr
				env = cell.Env
				continue
			}
		case ast.IfNode:
			conts = append(conts, cont{kind: kIfCond, mark: len(*pending), node: node, env: env})
			n = node.Cond
			continue
		case ast.IfZeroNode:
			conts = append(conts, cont{kind: kIfZeroCond, mark: len(*pending), node: node, env: env})
			n = node.Cond
			continue
		case ast.IfNilNode:
			conts = append(conts, cont{kind: kIfNilCond, mark: len(*pending), node: node, env: env})
			n = node.Cond
			continue
		case ast.AppendNode:
			conts = append(conts, cont{kind: kAppendLeft, mark: len(*pending), node: node.Right, env: env})
			n = node.Left
			continue
		case ast.ZFNode:
			n = evalZF(env, node.Body, node.Quals)
			continue
		case ast.ZFGeneratorNode:
			n = stepZFGenerator(node)
			continue
		case ast.RangeNode:
			v1 := Whnf(env, node.Start)
			v2 := Whnf(env, node.End)
			i1, ok1 := v1.(ast.IntNode)
			i2, ok2 := v2.(ast.IntNode)
			if !ok1 || !ok2 {
				panic(ast.RuntimeError{Msg: "Range bounds must evaluate to integers"})
			}
			if i1.Val > i2.Val {
				v = ast.NilNode{}
				break
			}
			nextRange := ast.RangeNode{Start: ast.IntNode{Val: i1.Val + 1}, End: v2}
			tPrime := ast.ThunkNode{Cell: &ast.ThunkCell{
				State: ast.Unevaluated,
				Expr:  nextRange,
				Env:   env,
			}}
			v = ast.ConsNode{Head: v1, Tail: tPrime}
		case ast.ProjNode:
			conts = append(conts, cont{kind: kProjTuple, mark: len(*pending), idx: node.Index, env: env})
			n = node.Tuple
			continue
		case ast.MatchErrorNode:
			panic(ast.RuntimeError{Msg: "Pattern matching exhausted"})
		case ast.AddNode:
			conts = append(conts, cont{kind: kBinopLeft, mark: len(*pending), node: node, val: node.Right, env: env})
			n = node.Left
			continue
		case ast.SubNode:
			conts = append(conts, cont{kind: kBinopLeft, mark: len(*pending), node: node, val: node.Right, env: env})
			n = node.Left
			continue
		case ast.MulNode:
			conts = append(conts, cont{kind: kBinopLeft, mark: len(*pending), node: node, val: node.Right, env: env})
			n = node.Left
			continue
		case ast.DivNode:
			conts = append(conts, cont{kind: kBinopLeft, mark: len(*pending), node: node, val: node.Right, env: env})
			n = node.Left
			continue
		case ast.ModNode:
			conts = append(conts, cont{kind: kBinopLeft, mark: len(*pending), node: node, val: node.Right, env: env})
			n = node.Left
			continue
		case ast.EqNode:
			conts = append(conts, cont{kind: kBinopLeft, mark: len(*pending), node: node, val: node.Right, env: env})
			n = node.Left
			continue
		case ast.NeNode:
			conts = append(conts, cont{kind: kBinopLeft, mark: len(*pending), node: node, val: node.Right, env: env})
			n = node.Left
			continue
		case ast.LtNode:
			conts = append(conts, cont{kind: kBinopLeft, mark: len(*pending), node: node, val: node.Right, env: env})
			n = node.Left
			continue
		case ast.GtNode:
			conts = append(conts, cont{kind: kBinopLeft, mark: len(*pending), node: node, val: node.Right, env: env})
			n = node.Left
			continue
		case ast.LeNode:
			conts = append(conts, cont{kind: kBinopLeft, mark: len(*pending), node: node, val: node.Right, env: env})
			n = node.Left
			continue
		case ast.GeNode:
			conts = append(conts, cont{kind: kBinopLeft, mark: len(*pending), node: node, val: node.Right, env: env})
			n = node.Left
			continue
		case ast.DiffNode:
			xs := Whnf(env, node.Left)
			ys := Whnf(env, node.Right)
			n = diff(env, xs, ys)
			continue
		case ast.AppNode:
			// fast paths for the common callee shapes skip the continuation
			switch lf := node.Left.(type) {
			case ast.VarNode:
				if lf.Name != "empty_map" && lf.Name != "empty_set" && resolverBuiltins[lf.Name] {
					n = applyBuiltin(env, lf.Name, node)
					continue
				}
			case ast.LamNode:
				argVal := bindArg(env, node.Right)
				env = env.Extend(lf.Var, argVal)
				n = lf.Body
				continue
			case ast.ClosureNode:
				argVal := bindArg(env, node.Right)
				env = lf.Env.Extend(lf.Var, argVal)
				n = lf.Body
				continue
			case ast.GlobalVarNode:
				if gv, gok := env.Globals[lf.Name]; gok {
					if lam, isLam := gv.(ast.LamNode); isLam {
						root := env.Root
						if root == nil {
							root = &ast.Env{Globals: env.Globals}
						}
						argVal := bindArg(env, node.Right)
						env = root.Extend(lam.Var, argVal)
						n = lam.Body
						continue
					}
				}
			case ast.LocalVarNode:
				curr := env
				for d := lf.Depth; d > 0 && curr != nil; d-- {
					curr = curr.Parent
				}
				if curr != nil {
					cv := curr.Val
					if th, isTh := cv.(ast.ThunkNode); isTh && th.Cell.State == ast.Evaluated {
						cv = th.Cell.Val
					}
					if cl, isCl := cv.(ast.ClosureNode); isCl {
						argVal := bindArg(env, node.Right)
						env = cl.Env.Extend(cl.Var, argVal)
						n = cl.Body
						continue
					}
				}
			}
			conts = append(conts, cont{kind: kAppFun, mark: len(*pending), node: node.Right, env: env})
			n = node.Left
			continue
		default:
			panic(fmt.Sprintf("Internal error: unhandled node type in whnf: %T", n))
		}

		// a value was produced: give it to the pending thunks recorded for
		// this operand, then resume the innermost continuation
		for {
			if len(conts) == 0 {
				return v
			}
			k := conts[len(conts)-1]
			conts = conts[:len(conts)-1]
			for _, cell := range (*pending)[k.mark:] {
				cell.Val = v
				cell.State = ast.Evaluated
			}
			*pending = (*pending)[:k.mark]
			switch k.kind {
			case kIfCond:
				ifn := k.node.(ast.IfNode)
				if b, ok := v.(ast.BoolNode); ok {
					if b.Val {
						n = ifn.Then
					} else {
						n = ifn.Else
					}
				} else if i, ok := v.(ast.IntNode); ok {
					if i.Val != 0 {
						n = ifn.Then
					} else {
						n = ifn.Else
					}
				} else {
					panic(ast.RuntimeError{Msg: fmt.Sprintf("If condition must be a boolean, got: %s", PrintNode(k.env, v))})
				}
				env = k.env
				continue machine
			case kIfZeroCond:
				ifn := k.node.(ast.IfZeroNode)
				i, ok := v.(ast.IntNode)
				if !ok {
					panic(ast.RuntimeError{Msg: "Condition must resolve to an integer"})
				}
				if i.Val == 0 {
					n = ifn.Then
				} else {
					n = ifn.Else
				}
				env = k.env
				continue machine
			case kIfNilCond:
				ifn := k.node.(ast.IfNilNode)
				switch v.(type) {
				case ast.NilNode:
					n = ifn.Then
				case ast.ConsNode:
					n = ifn.Else
				default:
					panic(ast.RuntimeError{Msg: "Condition must resolve to a list"})
				}
				env = k.env
				continue machine
			case kBinopLeft:
				right := k.val
				conts = append(conts, cont{kind: kBinopRight, mark: len(*pending), node: k.node, val: v, env: k.env})
				n = right
				env = k.env
				continue machine
			case kBinopRight:
				v = evalBinop(k.env, k.node, k.val, v)
			case kAppFun:
				switch f := v.(type) {
				case ast.ClosureNode:
					env = f.Env.Extend(f.Var, bindArg(k.env, k.node))
					n = f.Body
				case ast.LamNode:
					env = k.env.Extend(f.Var, bindArg(k.env, k.node))
					n = f.Body
				case ast.VarNode:
					n = applyBuiltin(k.env, f.Name, ast.AppNode{Left: f, Right: k.node})
					env = k.env
				default:
					res := applyPartial(k.env, v, ast.AppNode{Left: v, Right: k.node})
					if res == nil {
						panic(ast.RuntimeError{Msg: "Non-functional application"})
					}
					n = res
					env = k.env
				}
				continue machine
			case kProjTuple:
				t, ok := v.(ast.TupleNode)
				if !ok {
					panic(ast.RuntimeError{Msg: "Proj expects a tuple"})
				}
				n = t.Elems[k.idx]
				env = k.env
				continue machine
			case kAppendLeft:
				switch l := v.(type) {
				case ast.NilNode:
					n = k.node
					env = k.env
					continue machine
				case ast.ConsNode:
					v = ast.ConsNode{Head: l.Head, Tail: ast.ThunkNode{Cell: &ast.ThunkCell{
						State: ast.Unevaluated,
						Expr:  ast.AppendNode{Left: l.Tail, Right: k.node},
						Env:   k.env,
					}}}
				default:
					panic(ast.RuntimeError{Msg: "Append expects lists"})
				}
			}
		}
	}
}

// stepZFGenerator advances a list-comprehension generator until it can
// yield an element (or exhausts its source) and returns the node the
// trampoline should evaluate next. Source elements that fail the pattern
// match or a directly following filter qualifier are skipped in this loop
// without allocating generator/append/conditional machinery for them, and
// a yielded element conses straight onto the advanced generator instead of
// going through a per-element AppendNode.
func stepZFGenerator(node ast.ZFGeneratorNode) ast.Node {
	src := node.Src
	for {
		srcVal := Whnf(node.ZFEnv, src)
		switch s := srcVal.(type) {
		case ast.NilNode:
			return ast.NilNode{}
		case ast.ConsNode:
			matchRes, matchOk := matchPattern(node.ZFEnv, node.Pat, s.Head)
			if !matchOk {
				src = s.Tail
				continue
			}
			extendedEnv := node.ZFEnv
			for _, b := range matchRes {
				extendedEnv = extendedEnv.Extend(b.Name, b.Val)
			}
			// evaluate filter qualifiers that directly follow this
			// generator eagerly; on failure move to the next element
			rest := node.Rest
			passed := true
			for len(rest) > 0 {
				fq, isFilter := rest[0].(ast.FilterQual)
				if !isFilter {
					break
				}
				if !isTrueNode(Whnf(extendedEnv, fq.Cond)) {
					passed = false
					break
				}
				rest = rest[1:]
			}
			if !passed {
				src = s.Tail
				continue
			}
			nextGen := ast.ZFGeneratorNode{
				Pat:   node.Pat,
				Rest:  node.Rest,
				Src:   s.Tail,
				Body:  node.Body,
				ZFEnv: node.ZFEnv,
			}
			if len(rest) == 0 {
				// only filters followed: yield exactly one element
				h := node.Body
				if needsThunkCons(h) {
					h = ast.ThunkNode{Cell: &ast.ThunkCell{State: ast.Unevaluated, Expr: node.Body, Env: extendedEnv}}
				}
				return ast.ConsNode{Head: h, Tail: nextGen}
			}
			// a nested generator follows: splice its results in front
			firstList := evalZF(extendedEnv, node.Body, rest)
			return ast.AppendNode{Left: firstList, Right: nextGen}
		default:
			panic(ast.RuntimeError{Msg: "Generator source must be a list"})
		}
	}
}

// applyPartial applies a curried builtin partial-application value to the
// argument of node. It lives outside whnfCore so its bulky locals do not
// enlarge the recursive evaluation frame. Returns nil when fVal is not a
// partial-application value (i.e. the application target is not callable);
// every handled case returns a non-nil node for the trampoline to finish
// reducing.
func applyPartial(env *ast.Env, fVal ast.Node, node ast.AppNode) ast.Node {
	switch f := fVal.(type) {
	case ast.HLookupPartialNode:
		key := getMapKey(env, node.Right)
		val, ok := f.Map[key]
		if !ok {
			panic(ast.RuntimeError{Msg: "h_lookup: key not found: " + key})
		}
		return val
	case ast.HInsertPartialNode1:
		key := getMapKey(env, node.Right)
		return ast.HInsertPartialNode2{Map: f.Map, Key: key}
	case ast.HInsertPartialNode2:
		val := Whnf(env, node.Right)
		newMap := make(map[string]ast.Node, len(f.Map)+1)
		for k, v := range f.Map {
			newMap[k] = v
		}
		newMap[f.Key] = val
		return ast.MapNode{Map: newMap}
	case ast.MemberPartialNode:
		key := getMapKey(env, node.Right)
		_, ok := f.Set[key]
		return ast.BoolNode{Val: ok}
	case ast.SeqPartialNode:
		return node.Right
	case ast.SplitPartialNode:
		s := getStringValue(env, node.Right)
		var res []string
		var current strings.Builder
		delims := f.Delims
		for _, r := range s {
			isDelim := false
			for _, d := range delims {
				if r == d {
					isDelim = true
					break
				}
			}
			if isDelim {
				if current.Len() > 0 {
					res = append(res, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(r)
			}
		}
		if current.Len() > 0 {
			res = append(res, current.String())
		}
		var listNode ast.Node = ast.NilNode{}
		for i := len(res) - 1; i >= 0; i-- {
			listNode = ast.ConsNode{Head: MakeStringNode(res[i]), Tail: listNode}
		}
		return listNode
	case ast.ListGetPartialNode:
		idxVal := Whnf(env, node.Right)
		idx := idxVal.(ast.IntNode).Val
		if idx < 0 || idx >= int64(len(f.List)) {
			panic(ast.RuntimeError{Msg: fmt.Sprintf("list_get: index out of bounds: %d (size %d)", idx, len(f.List))})
		}
		return ast.IntNode{Val: f.List[idx]}
	case ast.ListSetPartialNode1:
		idxVal := Whnf(env, node.Right)
		idx := idxVal.(ast.IntNode).Val
		return ast.ListSetPartialNode2{List: f.List, Index: idx}
	case ast.ListSetPartialNode2:
		valNode := Whnf(env, node.Right)
		val := valNode.(ast.IntNode).Val
		idx := f.Index
		if idx < 0 || idx >= int64(len(f.List)) {
			panic(ast.RuntimeError{Msg: fmt.Sprintf("list_set: index out of bounds: %d (size %d)", idx, len(f.List))})
		}
		newList := make([]int64, len(f.List))
		copy(newList, f.List)
		newList[idx] = val
		var listNode ast.Node = ast.NilNode{}
		for i := len(newList) - 1; i >= 0; i-- {
			listNode = ast.ConsNode{Head: ast.IntNode{Val: newList[i]}, Tail: listNode}
		}
		return listNode
	case ast.MemoizeNode:
		argVal := Whnf(env, node.Right)
		key := PrintNode(env, argVal)
		if val, ok := f.Cache[key]; ok {
			return val
		}
		res := Whnf(env, ast.AppNode{Left: f.Func, Right: argVal})
		f.Cache[key] = res
		return res
	case ast.SortByPartialNode:
		listVal := Whnf(env, node.Right)
		var elems []ast.Node
		curr := listVal
		for {
			lVal := Whnf(env, curr)
			if cons, ok := lVal.(ast.ConsNode); ok {
				elems = append(elems, cons.Head)
				curr = cons.Tail
			} else if _, ok := lVal.(ast.NilNode); ok {
				break
			} else {
				panic(ast.RuntimeError{Msg: "sort_by expects a list"})
			}
		}
		var sortErr error
		slices.SortFunc(elems, func(a, b ast.Node) int {
			if sortErr != nil {
				return 0
			}
			res1 := Whnf(env, ast.AppNode{Left: f.Cmp, Right: a})
			res2 := Whnf(env, ast.AppNode{Left: res1, Right: b})
			if i, ok := res2.(ast.IntNode); ok {
				return int(i.Val)
			}
			sortErr = fmt.Errorf("sort_by: comparison function did not return an integer")
			return 0
		})
		if sortErr != nil {
			panic(ast.RuntimeError{Msg: sortErr.Error()})
		}
		var listNode ast.Node = ast.NilNode{}
		for i := len(elems) - 1; i >= 0; i-- {
			listNode = ast.ConsNode{Head: elems[i], Tail: listNode}
		}
		return listNode
	case ast.HLookupDefPartialNode1:
		key := getMapKey(env, node.Right)
		return ast.HLookupDefPartialNode2{Map: f.Map, Key: key}
	case ast.HLookupDefPartialNode2:
		valNode := Whnf(env, node.Right)
		val, ok := f.Map[f.Key]
		if !ok {
			return valNode
		}
		return val
	}
	return nil
}

// applyBuiltin applies the named builtin to the argument of node (kept out
// of whnfCore for the same stack-frame reason as applyPartial). The result
// is handed back to the trampoline, which finishes reducing it to WHNF.
func applyBuiltin(env *ast.Env, name string, node ast.AppNode) ast.Node {
	switch name {
	case "hd":
		e2Val := Whnf(env, node.Right)
		if c, ok := e2Val.(ast.ConsNode); ok {
			return c.Head
		}
		if _, ok := e2Val.(ast.NilNode); ok {
			panic(ast.RuntimeError{Msg: "hd applied to empty list"})
		}
		panic(ast.RuntimeError{Msg: "hd expects a list"})
	case "tl":
		e2Val := Whnf(env, node.Right)
		if c, ok := e2Val.(ast.ConsNode); ok {
			return c.Tail
		}
		if _, ok := e2Val.(ast.NilNode); ok {
			panic(ast.RuntimeError{Msg: "tl applied to empty list"})
		}
		panic(ast.RuntimeError{Msg: "tl expects a list"})
	case "read":
		filename := getStringValue(env, node.Right)
		content, err := os.ReadFile(filename)
		if err != nil {
			panic(ast.RuntimeError{Msg: fmt.Sprintf("Failed to read file: %s", filename)})
		}
		return MakeStringNode(string(content))
	case "lines":
		content := getStringValue(env, node.Right)
		strList := splitLines(content)
		var listNode ast.Node = ast.NilNode{}
		for i := len(strList) - 1; i >= 0; i-- {
			listNode = ast.ConsNode{Head: MakeStringNode(strList[i]), Tail: listNode}
		}
		return listNode
	case "numval":
		s := getStringValue(env, node.Right)
		sTrimmed := strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, s)
		v, err := strconv.ParseInt(sTrimmed, 10, 64)
		if err != nil {
			panic(ast.RuntimeError{Msg: "numval: invalid integer: " + s})
		}
		return ast.IntNode{Val: v}
	case "show":
		evaluatedNode := Whnf(env, node.Right)
		s := PrintNode(env, evaluatedNode)
		return MakeStringNode(s)
	case "length":
		var length int64 = 0
		curr := node.Right
		for {
			lVal := Whnf(env, curr)
			if cons, ok := lVal.(ast.ConsNode); ok {
				length++
				curr = cons.Tail
			} else if _, ok := lVal.(ast.NilNode); ok {
				break
			} else {
				panic(ast.RuntimeError{Msg: "length expects a list"})
			}
		}
		return ast.IntNode{Val: length}
	case "seq":
		Whnf(env, node.Right)
		return ast.SeqPartialNode{}
	case "reverse":
		var reversed ast.Node = ast.NilNode{}
		curr := node.Right
		for {
			lVal := Whnf(env, curr)
			if cons, ok := lVal.(ast.ConsNode); ok {
				reversed = ast.ConsNode{Head: cons.Head, Tail: reversed}
				curr = cons.Tail
			} else if _, ok := lVal.(ast.NilNode); ok {
				break
			} else {
				panic(ast.RuntimeError{Msg: "reverse expects a list"})
			}
		}
		return reversed
	case "h_lookup":
		mapVal := Whnf(env, node.Right)
		mNode, ok := mapVal.(ast.MapNode)
		if !ok {
			panic(ast.RuntimeError{Msg: "h_lookup: expected map as first argument"})
		}
		return ast.HLookupPartialNode{Map: mNode.Map}
	case "h_insert":
		mapVal := Whnf(env, node.Right)
		mNode, ok := mapVal.(ast.MapNode)
		if !ok {
			panic(ast.RuntimeError{Msg: "h_insert: expected map as first argument"})
		}
		return ast.HInsertPartialNode1{Map: mNode.Map}
	case "member":
		setVal := Whnf(env, node.Right)
		sNode, ok := setVal.(ast.SetNode)
		if !ok {
			panic(ast.RuntimeError{Msg: "member: expected set as first argument"})
		}
		return ast.MemberPartialNode{Set: sNode.Set}
	case "split":
		delims := getStringValue(env, node.Right)
		return ast.SplitPartialNode{Delims: delims}
	case "parse_ints":
		s := getStringValue(env, node.Right)
		var res []int64
		var current strings.Builder
		for _, r := range s {
			if (r >= '0' && r <= '9') || r == '-' {
				current.WriteRune(r)
			} else {
				if current.Len() > 0 {
					str := current.String()
					if str != "-" {
						val, err := strconv.ParseInt(str, 10, 64)
						if err == nil {
							res = append(res, val)
						}
					}
					current.Reset()
				}
			}
		}
		if current.Len() > 0 {
			str := current.String()
			if str != "-" {
				val, err := strconv.ParseInt(str, 10, 64)
				if err == nil {
					res = append(res, val)
				}
			}
		}
		var listNode ast.Node = ast.NilNode{}
		for i := len(res) - 1; i >= 0; i-- {
			listNode = ast.ConsNode{Head: ast.IntNode{Val: res[i]}, Tail: listNode}
		}
		return listNode
	case "list_get":
		listVal := Whnf(env, node.Right)
		slice := getIntSlice(env, listVal)
		return ast.ListGetPartialNode{List: slice}
	case "list_set":
		listVal := Whnf(env, node.Right)
		slice := getIntSlice(env, listVal)
		return ast.ListSetPartialNode1{List: slice}
	case "memoize":
		fn := Whnf(env, node.Right)
		return ast.MemoizeNode{Func: fn, Cache: make(map[string]ast.Node)}
	case "sort_ints":
		listVal := Whnf(env, node.Right)
		var elems []int64
		curr := listVal
		for {
			lVal := Whnf(env, curr)
			if cons, ok := lVal.(ast.ConsNode); ok {
				val := Whnf(env, cons.Head).(ast.IntNode).Val
				elems = append(elems, val)
				curr = cons.Tail
			} else if _, ok := lVal.(ast.NilNode); ok {
				break
			} else {
				panic(ast.RuntimeError{Msg: "sort_ints expects a list of integers"})
			}
		}
		slices.Sort(elems)
		var listNode ast.Node = ast.NilNode{}
		for i := len(elems) - 1; i >= 0; i-- {
			listNode = ast.ConsNode{Head: ast.IntNode{Val: elems[i]}, Tail: listNode}
		}
		return listNode
	case "sort_edges":
		listVal := Whnf(env, node.Right)
		// decorate-sort-undecorate: force each element's key exactly once
		// instead of re-forcing thunks in every comparison
		var elems []keyedNode
		curr := listVal
		for {
			lVal := Whnf(env, curr)
			if cons, ok := lVal.(ast.ConsNode); ok {
				t := Whnf(env, cons.Head).(ast.TupleNode)
				key := Whnf(env, t.Elems[2]).(ast.IntNode).Val
				elems = append(elems, keyedNode{Key: key, Node: cons.Head})
				curr = cons.Tail
			} else if _, ok := lVal.(ast.NilNode); ok {
				break
			} else {
				panic(ast.RuntimeError{Msg: "sort_edges expects a list"})
			}
		}
		return sortKeyedToList(elems)
	case "sort_pts":
		listVal := Whnf(env, node.Right)
		var elems []keyedNode
		curr := listVal
		for {
			lVal := Whnf(env, curr)
			if cons, ok := lVal.(ast.ConsNode); ok {
				t := Whnf(env, cons.Head).(ast.TupleNode)
				coords := Whnf(env, t.Elems[1]).(ast.TupleNode)
				key := Whnf(env, coords.Elems[0]).(ast.IntNode).Val
				elems = append(elems, keyedNode{Key: key, Node: cons.Head})
				curr = cons.Tail
			} else if _, ok := lVal.(ast.NilNode); ok {
				break
			} else {
				panic(ast.RuntimeError{Msg: "sort_pts expects a list"})
			}
		}
		return sortKeyedToList(elems)
	case "sort_by":
		cmpNode := Whnf(env, node.Right)
		return ast.SortByPartialNode{Cmp: cmpNode}
	case "h_lookup_def":
		mapVal := Whnf(env, node.Right)
		mNode, ok := mapVal.(ast.MapNode)
		if !ok {
			panic(ast.RuntimeError{Msg: "h_lookup_def: expected map as first argument"})
		}
		return ast.HLookupDefPartialNode1{Map: mNode.Map}
	default:
		panic(ast.RuntimeError{Msg: "Unbound variable: " + name})
	}
}

func escapeChar(r rune) string {
	switch r {
	case '\n':
		return "\\n"
	case '\t':
		return "\\t"
	case '\'':
		return "\\'"
	case '\\':
		return "\\\\"
	default:
		return string(r)
	}
}

func escapeString(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '\n':
			sb.WriteString("\\n")
		case '\t':
			sb.WriteString("\\t")
		case '"':
			sb.WriteString("\\\"")
		case '\\':
			sb.WriteString("\\\\")
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func PrintNode(env *ast.Env, n ast.Node) string {
	switch node := n.(type) {
	case ast.IntNode:
		return strconv.FormatInt(node.Val, 10)
	case ast.BoolNode:
		if node.Val {
			return "True"
		}
		return "False"
	case ast.CharNode:
		return "'" + escapeChar(node.Val) + "'"
	case ast.NilNode:
		return "[]"
	case ast.MapNode:
		return "<map>"
	case ast.SetNode:
		return "<set>"
	case ast.HLookupPartialNode:
		return "<h_lookup partial>"
	case ast.HInsertPartialNode1:
		return "<h_insert partial 1>"
	case ast.HInsertPartialNode2:
		return "<h_insert partial 2>"
	case ast.MemberPartialNode:
		return "<member partial>"
	case ast.SeqPartialNode:
		return "<seq partial>"
	case ast.SplitPartialNode:
		return "<split partial>"
	case ast.ListGetPartialNode:
		return "<list_get partial>"
	case ast.ListSetPartialNode1:
		return "<list_set partial 1>"
	case ast.ListSetPartialNode2:
		return "<list_set partial 2>"
	case ast.MemoizeNode:
		return "<memoized>"
	case ast.SortByPartialNode:
		return "<sort_by partial>"
	case ast.HLookupDefPartialNode1:
		return "<h_lookup_def partial 1>"
	case ast.HLookupDefPartialNode2:
		return "<h_lookup_def partial 2>"
	case ast.LamNode:
		return "\\" + node.Var + ". <closure>"
	case ast.ClosureNode:
		return "\\" + node.Var + ". <closure>"
	case ast.LetNode:
		return "<let>"
	case ast.VarNode:
		return node.Name
	case ast.LocalVarNode:
		return node.Name
	case ast.GlobalVarNode:
		return node.Name
	case ast.AppNode:
		return "(" + PrintNode(env, node.Left) + " " + PrintNode(env, node.Right) + ")"
	case ast.SubNode:
		return "(" + PrintNode(env, node.Left) + " - " + PrintNode(env, node.Right) + ")"
	case ast.AddNode:
		return "(" + PrintNode(env, node.Left) + " + " + PrintNode(env, node.Right) + ")"
	case ast.MulNode:
		return "(" + PrintNode(env, node.Left) + " * " + PrintNode(env, node.Right) + ")"
	case ast.DivNode:
		return "(" + PrintNode(env, node.Left) + " / " + PrintNode(env, node.Right) + ")"
	case ast.DiffNode:
		return "(" + PrintNode(env, node.Left) + " -- " + PrintNode(env, node.Right) + ")"
	case ast.EqNode:
		return "(" + PrintNode(env, node.Left) + " == " + PrintNode(env, node.Right) + ")"
	case ast.NeNode:
		return "(" + PrintNode(env, node.Left) + " != " + PrintNode(env, node.Right) + ")"
	case ast.LtNode:
		return "(" + PrintNode(env, node.Left) + " < " + PrintNode(env, node.Right) + ")"
	case ast.GtNode:
		return "(" + PrintNode(env, node.Left) + " > " + PrintNode(env, node.Right) + ")"
	case ast.LeNode:
		return "(" + PrintNode(env, node.Left) + " <= " + PrintNode(env, node.Right) + ")"
	case ast.GeNode:
		return "(" + PrintNode(env, node.Left) + " >= " + PrintNode(env, node.Right) + ")"
	case ast.ModNode:
		return "(" + PrintNode(env, node.Left) + " mod " + PrintNode(env, node.Right) + ")"
	case ast.TupleNode:
		var elms []string
		for _, e := range node.Elems {
			elms = append(elms, PrintNode(env, Whnf(env, e)))
		}
		return "(" + strings.Join(elms, ",") + ")"
	case ast.IfZeroNode, ast.IfNode:
		return "<conditional>"
	case ast.IfNilNode:
		return "<conditional-nil>"
	case ast.AppendNode:
		return "<append>"
	case ast.ZFNode:
		return "<zf-comprehension>"
	case ast.ZFGeneratorNode:
		return "<zf-generator>"
	case ast.MatchErrorNode:
		return "<match-error>"
	case ast.ThunkNode:
		return "<thunk>"
	case ast.RangeNode:
		return "[" + PrintNode(env, node.Start) + ".." + PrintNode(env, node.End) + "]"
	case ast.ConsNode:
		if s, isStr := IsString(env, node); isStr {
			if s == "" {
				return "[]"
			}
			return "\"" + escapeString(s) + "\""
		}
		var elms []string
		curr := ast.Node(node)
		for {
			currVal := Whnf(env, curr)
			if cons, ok := currVal.(ast.ConsNode); ok {
				elms = append(elms, PrintNode(env, Whnf(env, cons.Head)))
				curr = cons.Tail
			} else if _, ok := currVal.(ast.NilNode); ok {
				break
			} else {
				elms = append(elms, PrintNode(env, currVal))
				break
			}
		}
		return "[" + strings.Join(elms, ",") + "]"
	case ast.ProjNode:
		return "<projection-" + strconv.Itoa(node.Index) + ">"
	}
	return "<unknown>"
}

func IsString(env *ast.Env, node ast.Node) (string, bool) {
	var sb strings.Builder
	curr := node
	for {
		v := Whnf(env, curr)
		switch val := v.(type) {
		case ast.NilNode:
			return sb.String(), true
		case ast.ConsNode:
			h := Whnf(env, val.Head)
			if c, ok := h.(ast.CharNode); ok {
				sb.WriteRune(c.Val)
				curr = val.Tail
			} else {
				return "", false
			}
		default:
			return "", false
		}
	}
}

func DebugPrintNode(n ast.Node) string {
	if n == nil {
		return "nil"
	}
	switch node := n.(type) {
	case ast.IntNode:
		return fmt.Sprintf("Int(%d)", node.Val)
	case ast.CharNode:
		return fmt.Sprintf("Char(%q)", node.Val)
	case ast.NilNode:
		return "Nil"
	case ast.MapNode:
		return "Map"
	case ast.SetNode:
		return "Set"
	case ast.HLookupPartialNode:
		return "HLookupPartial"
	case ast.HInsertPartialNode1:
		return "HInsertPartial1"
	case ast.HInsertPartialNode2:
		return "HInsertPartial2"
	case ast.MemberPartialNode:
		return "MemberPartial"
	case ast.SeqPartialNode:
		return "SeqPartial"
	case ast.SplitPartialNode:
		return "SplitPartial"
	case ast.ListGetPartialNode:
		return "ListGetPartial"
	case ast.ListSetPartialNode1:
		return "ListSetPartial1"
	case ast.ListSetPartialNode2:
		return "ListSetPartial2"
	case ast.MemoizeNode:
		return "Memoize"
	case ast.SortByPartialNode:
		return "SortByPartial"
	case ast.HLookupDefPartialNode1:
		return "HLookupDef1"
	case ast.HLookupDefPartialNode2:
		return "HLookupDef2"
	case ast.VarNode:
		return fmt.Sprintf("Var(%s)", node.Name)
	case ast.LocalVarNode:
		return fmt.Sprintf("LocalVar(%s,%d)", node.Name, node.Depth)
	case ast.GlobalVarNode:
		return fmt.Sprintf("GlobalVar(%s)", node.Name)
	case ast.LamNode:
		return fmt.Sprintf("Lam(%s, %s)", node.Var, DebugPrintNode(node.Body))
	case ast.AppNode:
		return fmt.Sprintf("App(%s, %s)", DebugPrintNode(node.Left), DebugPrintNode(node.Right))
	case ast.ConsNode:
		return fmt.Sprintf("Cons(%s, %s)", DebugPrintNode(node.Head), DebugPrintNode(node.Tail))
	case ast.TupleNode:
		var elms []string
		for _, e := range node.Elems {
			elms = append(elms, DebugPrintNode(e))
		}
		return fmt.Sprintf("Tuple(%s)", strings.Join(elms, ", "))
	case ast.LetNode:
		var binds []string
		for _, b := range node.Bindings {
			binds = append(binds, fmt.Sprintf("%s=%s", b.Name, DebugPrintNode(b.Expr)))
		}
		return fmt.Sprintf("Let([%s], %s)", strings.Join(binds, "; "), DebugPrintNode(node.Body))
	case ast.IfNode:
		return fmt.Sprintf("If(%s, %s, %s)", DebugPrintNode(node.Cond), DebugPrintNode(node.Then), DebugPrintNode(node.Else))
	case ast.IfZeroNode:
		return fmt.Sprintf("IfZero(%s, %s, %s)", DebugPrintNode(node.Cond), DebugPrintNode(node.Then), DebugPrintNode(node.Else))
	case ast.IfNilNode:
		return fmt.Sprintf("IfNil(%s, %s, %s)", DebugPrintNode(node.Cond), DebugPrintNode(node.Then), DebugPrintNode(node.Else))
	case ast.AddNode:
		return fmt.Sprintf("Add(%s, %s)", DebugPrintNode(node.Left), DebugPrintNode(node.Right))
	case ast.SubNode:
		return fmt.Sprintf("Sub(%s, %s)", DebugPrintNode(node.Left), DebugPrintNode(node.Right))
	case ast.MulNode:
		return fmt.Sprintf("Mul(%s, %s)", DebugPrintNode(node.Left), DebugPrintNode(node.Right))
	case ast.DivNode:
		return fmt.Sprintf("Div(%s, %s)", DebugPrintNode(node.Left), DebugPrintNode(node.Right))
	case ast.ModNode:
		return fmt.Sprintf("Mod(%s, %s)", DebugPrintNode(node.Left), DebugPrintNode(node.Right))
	case ast.EqNode:
		return fmt.Sprintf("Eq(%s, %s)", DebugPrintNode(node.Left), DebugPrintNode(node.Right))
	case ast.NeNode:
		return fmt.Sprintf("Ne(%s, %s)", DebugPrintNode(node.Left), DebugPrintNode(node.Right))
	case ast.LtNode:
		return fmt.Sprintf("Lt(%s, %s)", DebugPrintNode(node.Left), DebugPrintNode(node.Right))
	case ast.GtNode:
		return fmt.Sprintf("Gt(%s, %s)", DebugPrintNode(node.Left), DebugPrintNode(node.Right))
	case ast.LeNode:
		return fmt.Sprintf("Le(%s, %s)", DebugPrintNode(node.Left), DebugPrintNode(node.Right))
	case ast.GeNode:
		return fmt.Sprintf("Ge(%s, %s)", DebugPrintNode(node.Left), DebugPrintNode(node.Right))
	case ast.ProjNode:
		return fmt.Sprintf("Proj(%d, %s)", node.Index, DebugPrintNode(node.Tuple))
	case ast.MatchErrorNode:
		return "MatchError"
	default:
		return fmt.Sprintf("%T", n)
	}
}

// keyedNode pairs a list element with its pre-extracted integer sort key so
// native sorts compare plain int64s instead of re-forcing thunks per
// comparison (decorate-sort-undecorate).
type keyedNode struct {
	Key  int64
	Node ast.Node
}

func sortKeyedToList(elems []keyedNode) ast.Node {
	slices.SortFunc(elems, func(a, b keyedNode) int {
		if a.Key < b.Key {
			return -1
		}
		if a.Key > b.Key {
			return 1
		}
		return 0
	})
	var listNode ast.Node = ast.NilNode{}
	for i := len(elems) - 1; i >= 0; i-- {
		listNode = ast.ConsNode{Head: elems[i].Node, Tail: listNode}
	}
	return listNode
}

func getIntSlice(env *ast.Env, n ast.Node) []int64 {
	var res []int64
	curr := n
	for {
		lVal := Whnf(env, curr)
		if cons, ok := lVal.(ast.ConsNode); ok {
			val := Whnf(env, cons.Head)
			if i, ok := val.(ast.IntNode); ok {
				res = append(res, i.Val)
			} else {
				panic(ast.RuntimeError{Msg: "list_get/list_set expects a list of integers"})
			}
			curr = cons.Tail
		} else if _, ok := lVal.(ast.NilNode); ok {
			break
		} else {
			panic(ast.RuntimeError{Msg: "list_get/list_set expects a list"})
		}
	}
	return res
}
