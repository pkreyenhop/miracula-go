package eval

import (
	"fmt"
	"os"
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

func smlDiv(a, b int) int {
	q := a / b
	r := a % b
	if (r > 0 && b < 0) || (r < 0 && b > 0) {
		q--
	}
	return q
}

func smlMod(a, b int) int {
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
	var collect func(ast.Node, []rune) string
	collect = func(current ast.Node, acc []rune) string {
		switch l := Whnf(env, current).(type) {
		case ast.NilNode:
			return string(acc)
		case ast.ConsNode:
			hVal := Whnf(env, l.Head)
			if c, ok := hVal.(ast.CharNode); ok {
				return collect(l.Tail, append(acc, c.Val))
			}
			panic(ast.RuntimeError{Msg: "Expected char in string"})
		default:
			panic(ast.RuntimeError{Msg: "Expected string"})
		}
	}
	return collect(node, []rune{})
}

func MakeStringNode(s string) ast.Node {
	runes := []rune(s)
	var makeList func([]rune) ast.Node
	makeList = func(rs []rune) ast.Node {
		if len(rs) == 0 {
			return ast.NilNode{}
		}
		return ast.ConsNode{
			Head: ast.CharNode{Val: rs[0]},
			Tail: makeList(rs[1:]),
		}
	}
	return makeList(runes)
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
			if isTrueNode(eqH) {
				eqT := Whnf(env, ast.EqNode{Left: x1.Tail, Right: x2.Tail})
				if isTrueNode(eqT) {
					return true
				}
			}
			return false
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

func Whnf(env *ast.Env, n ast.Node) ast.Node {
	if IsInterrupted() {
		panic(InterruptedException{})
	}
	for {
		switch node := n.(type) {
		case ast.IntNode:
			return node
		case ast.BoolNode:
			return node
		case ast.CharNode:
			return node
		case ast.NilNode:
			return node
		case ast.LamNode:
			return ast.ClosureNode{Var: node.Var, Body: node.Body, Env: env}
		case ast.ClosureNode:
			return node
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
			return ast.ConsNode{Head: h, Tail: t}
		case ast.TupleNode:
			elmsPrime := make([]ast.Node, len(node.Elems))
			for i, e := range node.Elems {
				if needsThunkTuple(e) {
					elmsPrime[i] = ast.ThunkNode{Cell: &ast.ThunkCell{State: ast.Unevaluated, Expr: e, Env: env}}
				} else {
					elmsPrime[i] = e
				}
			}
			return ast.TupleNode{Elems: elmsPrime}
		case ast.VarNode:
			name := node.Name
			switch name {
			case "hd", "tl", "show", "read", "lines", "numval", "length", "reverse":
				return node
			}

			var val ast.Node
			var ok bool
			if env.Name == name {
				val = env.Val
				ok = true
			} else if env.Parent != nil && env.Parent.Name == name {
				val = env.Parent.Val
				ok = true
			} else {
				val, ok = env.Lookup(name)
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
						return cv
					default:
						n = cv
						continue
					}
				case ast.Evaluating:
					panic(ast.BlackholeError{Msg: "Infinite loop on identifier: " + name})
				case ast.Unevaluated:
					cell.State = ast.Evaluating
					res := Whnf(cell.Env, cell.Expr)
					cell.State = ast.Evaluated
					cell.Val = res
					switch cv := res.(type) {
					case ast.IntNode, ast.BoolNode, ast.CharNode, ast.NilNode, ast.ClosureNode, ast.MatchErrorNode:
						return cv
					default:
						n = cv
						continue
					}
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
					return cv
				default:
					n = cv
					continue
				}
			case ast.Evaluating:
				panic(ast.BlackholeError{Msg: "Infinite loop inside generic thunk node"})
			case ast.Unevaluated:
				cell.State = ast.Evaluating
				res := Whnf(cell.Env, cell.Expr)
				cell.State = ast.Evaluated
				cell.Val = res
				switch cv := res.(type) {
				case ast.IntNode, ast.BoolNode, ast.CharNode, ast.NilNode, ast.ClosureNode, ast.MatchErrorNode:
					return cv
				default:
					n = cv
					continue
				}
			}
		case ast.IfNode:
			condVal := Whnf(env, node.Cond)
			if b, ok := condVal.(ast.BoolNode); ok {
				if b.Val {
					n = node.Then
				} else {
					n = node.Else
				}
				continue
			}
			if i, ok := condVal.(ast.IntNode); ok {
				if i.Val != 0 {
					n = node.Then
				} else {
					n = node.Else
				}
				continue
			}
			panic(ast.RuntimeError{Msg: fmt.Sprintf("If condition must be a boolean, got: %s", PrintNode(env, condVal))})
		case ast.IfZeroNode:
			condVal := Whnf(env, node.Cond)
			if i, ok := condVal.(ast.IntNode); ok {
				if i.Val == 0 {
					n = node.Then
				} else {
					n = node.Else
				}
				continue
			}
			panic(ast.RuntimeError{Msg: "Condition must resolve to an integer"})
		case ast.IfNilNode:
			condVal := Whnf(env, node.Cond)
			switch condVal.(type) {
			case ast.NilNode:
				n = node.Then
				continue
			case ast.ConsNode:
				n = node.Else
				continue
			default:
				panic(ast.RuntimeError{Msg: "Condition must resolve to a list"})
			}
		case ast.AppendNode:
			e1Val := Whnf(env, node.Left)
			switch l := e1Val.(type) {
			case ast.NilNode:
				n = node.Right
				continue
			case ast.ConsNode:
				tPrime := ast.ThunkNode{Cell: &ast.ThunkCell{
					State: ast.Unevaluated,
					Expr:  ast.AppendNode{Left: l.Tail, Right: node.Right},
					Env:   env,
				}}
				return ast.ConsNode{Head: l.Head, Tail: tPrime}
			default:
				panic(ast.RuntimeError{Msg: "Append expects lists"})
			}
		case ast.ZFNode:
			n = evalZF(env, node.Body, node.Quals)
			continue
		case ast.ZFGeneratorNode:
			srcVal := Whnf(node.ZFEnv, node.Src)
			switch s := srcVal.(type) {
			case ast.NilNode:
				return ast.NilNode{}
			case ast.ConsNode:
				matchRes, matchOk := matchPattern(node.ZFEnv, node.Pat, s.Head)
				nextGen := ast.ZFGeneratorNode{
					Pat:   node.Pat,
					Rest:  node.Rest,
					Src:   s.Tail,
					Body:  node.Body,
					ZFEnv: node.ZFEnv,
				}
				if matchOk {
					extendedEnv := node.ZFEnv
					for _, b := range matchRes {
						extendedEnv = extendedEnv.Extend(b.Name, b.Val)
					}
					firstList := evalZF(extendedEnv, node.Body, node.Rest)
					n = ast.AppendNode{Left: firstList, Right: nextGen}
					continue
				} else {
					n = nextGen
					continue
				}
			default:
				panic(ast.RuntimeError{Msg: "Generator source must be a list"})
			}
		case ast.RangeNode:
			v1 := Whnf(env, node.Start)
			v2 := Whnf(env, node.End)
			i1, ok1 := v1.(ast.IntNode)
			i2, ok2 := v2.(ast.IntNode)
			if !ok1 || !ok2 {
				panic(ast.RuntimeError{Msg: "Range bounds must evaluate to integers"})
			}
			if i1.Val > i2.Val {
				return ast.NilNode{}
			}
			nextRange := ast.RangeNode{Start: ast.IntNode{Val: i1.Val + 1}, End: v2}
			tPrime := ast.ThunkNode{Cell: &ast.ThunkCell{
				State: ast.Unevaluated,
				Expr:  nextRange,
				Env:   env,
			}}
			return ast.ConsNode{Head: v1, Tail: tPrime}
		case ast.ProjNode:
			tplVal := Whnf(env, node.Tuple)
			if t, ok := tplVal.(ast.TupleNode); ok {
				n = t.Elems[node.Index]
				continue
			}
			panic(ast.RuntimeError{Msg: "Proj expects a tuple"})
		case ast.MatchErrorNode:
			panic(ast.RuntimeError{Msg: "Pattern matching exhausted"})
		case ast.AddNode:
			v1 := Whnf(env, node.Left)
			v2 := Whnf(env, node.Right)
			i1, ok1 := v1.(ast.IntNode)
			i2, ok2 := v2.(ast.IntNode)
			if !ok1 || !ok2 {
				panic(ast.RuntimeError{Msg: "Addition expects integers"})
			}
			return ast.IntNode{Val: i1.Val + i2.Val}
		case ast.SubNode:
			v1 := Whnf(env, node.Left)
			v2 := Whnf(env, node.Right)
			i1, ok1 := v1.(ast.IntNode)
			i2, ok2 := v2.(ast.IntNode)
			if !ok1 || !ok2 {
				panic(ast.RuntimeError{Msg: "Subtraction expects integers"})
			}
			return ast.IntNode{Val: i1.Val - i2.Val}
		case ast.MulNode:
			v1 := Whnf(env, node.Left)
			v2 := Whnf(env, node.Right)
			i1, ok1 := v1.(ast.IntNode)
			i2, ok2 := v2.(ast.IntNode)
			if !ok1 || !ok2 {
				panic(ast.RuntimeError{Msg: "Multiplication expects integers"})
			}
			return ast.IntNode{Val: i1.Val * i2.Val}
		case ast.DivNode:
			v1 := Whnf(env, node.Left)
			v2 := Whnf(env, node.Right)
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
			v1 := Whnf(env, node.Left)
			v2 := Whnf(env, node.Right)
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
			v1 := Whnf(env, node.Left)
			v2 := Whnf(env, node.Right)
			return ast.BoolNode{Val: eq(env, v1, v2)}
		case ast.NeNode:
			v1 := Whnf(env, node.Left)
			v2 := Whnf(env, node.Right)
			return ast.BoolNode{Val: !eq(env, v1, v2)}
		case ast.LtNode:
			v1 := Whnf(env, node.Left)
			v2 := Whnf(env, node.Right)
			i1, ok1 := v1.(ast.IntNode)
			i2, ok2 := v2.(ast.IntNode)
			if !ok1 || !ok2 {
				panic(ast.RuntimeError{Msg: "Less-than expects integers"})
			}
			return ast.BoolNode{Val: i1.Val < i2.Val}
		case ast.GtNode:
			v1 := Whnf(env, node.Left)
			v2 := Whnf(env, node.Right)
			i1, ok1 := v1.(ast.IntNode)
			i2, ok2 := v2.(ast.IntNode)
			if !ok1 || !ok2 {
				panic(ast.RuntimeError{Msg: "Greater-than expects integers"})
			}
			return ast.BoolNode{Val: i1.Val > i2.Val}
		case ast.LeNode:
			v1 := Whnf(env, node.Left)
			v2 := Whnf(env, node.Right)
			i1, ok1 := v1.(ast.IntNode)
			i2, ok2 := v2.(ast.IntNode)
			if !ok1 || !ok2 {
				panic(ast.RuntimeError{Msg: "Less-than-or-equal expects integers"})
			}
			return ast.BoolNode{Val: i1.Val <= i2.Val}
		case ast.GeNode:
			v1 := Whnf(env, node.Left)
			v2 := Whnf(env, node.Right)
			i1, ok1 := v1.(ast.IntNode)
			i2, ok2 := v2.(ast.IntNode)
			if !ok1 || !ok2 {
				panic(ast.RuntimeError{Msg: "Greater-than-or-equal expects integers"})
			}
			return ast.BoolNode{Val: i1.Val >= i2.Val}
		case ast.DiffNode:
			xs := Whnf(env, node.Left)
			ys := Whnf(env, node.Right)
			n = diff(env, xs, ys)
			continue
		case ast.AppNode:
			fVal := Whnf(env, node.Left)
			switch f := fVal.(type) {
			case ast.VarNode:
				switch f.Name {
				case "hd":
					e2Val := Whnf(env, node.Right)
					if c, ok := e2Val.(ast.ConsNode); ok {
						n = c.Head
						continue
					}
					if _, ok := e2Val.(ast.NilNode); ok {
						panic(ast.RuntimeError{Msg: "hd applied to empty list"})
					}
					panic(ast.RuntimeError{Msg: "hd expects a list"})
				case "tl":
					e2Val := Whnf(env, node.Right)
					if c, ok := e2Val.(ast.ConsNode); ok {
						n = c.Tail
						continue
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
					var makeNodeList func(strs []string) ast.Node
					makeNodeList = func(strs []string) ast.Node {
						if len(strs) == 0 {
							return ast.NilNode{}
						}
						return ast.ConsNode{
							Head: MakeStringNode(strs[0]),
							Tail: makeNodeList(strs[1:]),
						}
					}
					return makeNodeList(strList)
				case "numval":
					s := getStringValue(env, node.Right)
					sTrimmed := strings.Map(func(r rune) rune {
						if unicode.IsSpace(r) {
							return -1
						}
						return r
					}, s)
					v, err := strconv.Atoi(sTrimmed)
					if err != nil {
						panic(ast.RuntimeError{Msg: "numval: invalid integer: " + s})
					}
					return ast.IntNode{Val: v}
				case "show":
					evaluatedNode := Whnf(env, node.Right)
					s := PrintNode(env, evaluatedNode)
					return MakeStringNode(s)
				case "length":
					length := 0
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
				default:
					panic(ast.RuntimeError{Msg: "Unbound variable: " + f.Name})
				}
			case ast.ClosureNode:
				var valNode ast.Node
				switch r := node.Right.(type) {
				case ast.IntNode, ast.CharNode, ast.NilNode, ast.ClosureNode, ast.ThunkNode, ast.MatchErrorNode:
					valNode = r
				case ast.LamNode:
					valNode = ast.ClosureNode{Var: r.Var, Body: r.Body, Env: env}
				default:
					valNode = ast.ThunkNode{Cell: &ast.ThunkCell{State: ast.Unevaluated, Expr: node.Right, Env: env}}
				}
				env = f.Env.Extend(f.Var, valNode)
				n = f.Body
				continue
			case ast.LamNode:
				var valNode ast.Node
				switch r := node.Right.(type) {
				case ast.IntNode, ast.CharNode, ast.NilNode, ast.ClosureNode, ast.ThunkNode, ast.MatchErrorNode:
					valNode = r
				case ast.LamNode:
					valNode = ast.ClosureNode{Var: r.Var, Body: r.Body, Env: env}
				default:
					valNode = ast.ThunkNode{Cell: &ast.ThunkCell{State: ast.Unevaluated, Expr: node.Right, Env: env}}
				}
				env = env.Extend(f.Var, valNode)
				n = f.Body
				continue
			default:
				panic(ast.RuntimeError{Msg: "Non-functional application"})
			}
		}
		panic(fmt.Sprintf("Internal error: unhandled node type in whnf: %T", n))
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
		return strconv.Itoa(node.Val)
	case ast.BoolNode:
		if node.Val {
			return "True"
		}
		return "False"
	case ast.CharNode:
		return "'" + escapeChar(node.Val) + "'"
	case ast.NilNode:
		return "[]"
	case ast.LamNode:
		return "\\" + node.Var + ". <closure>"
	case ast.ClosureNode:
		return "\\" + node.Var + ". <closure>"
	case ast.LetNode:
		return "<let>"
	case ast.VarNode:
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
	case ast.VarNode:
		return fmt.Sprintf("Var(%s)", node.Name)
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
