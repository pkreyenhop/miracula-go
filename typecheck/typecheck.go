package typecheck

import (
	"fmt"
	"pkreyenhop.com/miracula-go/ast"
	"strings"
)

type Type interface {
	String() string
}

type PrimType int

const (
	TInt PrimType = iota
	TBool
	TChar
)

func (t PrimType) String() string {
	switch t {
	case TInt:
		return "Int"
	case TBool:
		return "Bool"
	case TChar:
		return "Char"
	}
	return "Unknown"
}

type VarType struct {
	Id int
}

func (t VarType) String() string {
	if t.Id < 26 {
		return string(rune('a' + t.Id))
	}
	return fmt.Sprintf("a%d", t.Id)
}

type FunType struct {
	From Type
	To   Type
}

func (t FunType) String() string {
	var fromStr string
	if _, ok := t.From.(FunType); ok {
		fromStr = "(" + t.From.String() + ")"
	} else {
		fromStr = t.From.String()
	}
	return fromStr + " -> " + t.To.String()
}

type ListType struct {
	Elem Type
}

func (t ListType) String() string {
	return "[" + t.Elem.String() + "]"
}

type TupleType struct {
	Elems []Type
}

func (t TupleType) String() string {
	var strs []string
	for _, e := range t.Elems {
		strs = append(strs, e.String())
	}
	return "(" + strings.Join(strs, ", ") + ")"
}

type MapType struct {
	Key Type
	Val Type
}

func (t MapType) String() string {
	return "Map(" + t.Key.String() + ", " + t.Val.String() + ")"
}

type SetType struct {
	Elem Type
}

func (t SetType) String() string {
	return "Set(" + t.Elem.String() + ")"
}

type VecType struct {
	Elem Type
}

func (t VecType) String() string {
	return "Vec(" + t.Elem.String() + ")"
}

// PQType is a priority queue whose values have type Elem (priorities are num).
type PQType struct {
	Elem Type
}

func (t PQType) String() string {
	return "PQ(" + t.Elem.String() + ")"
}

type Scheme struct {
	Vars []int
	Ty   Type
}

func (s Scheme) String() string {
	if len(s.Vars) == 0 {
		return s.Ty.String()
	}
	var vars []string
	for _, v := range s.Vars {
		vars = append(vars, VarType{Id: v}.String())
	}
	return "forall " + strings.Join(vars, " ") + ". " + s.Ty.String()
}

type TypeEnv struct {
	Parent *TypeEnv
	Map    map[string]Scheme
}

func NewTypeEnv(parent *TypeEnv) *TypeEnv {
	return &TypeEnv{
		Parent: parent,
		Map:    make(map[string]Scheme),
	}
}

func (env *TypeEnv) Lookup(x string) (Scheme, bool) {
	for curr := env; curr != nil; curr = curr.Parent {
		if s, ok := curr.Map[x]; ok {
			return s, true
		}
	}
	return Scheme{}, false
}

func (env *TypeEnv) Extend(x string, s Scheme) *TypeEnv {
	next := NewTypeEnv(env.Parent)
	for k, v := range env.Map {
		next.Map[k] = v
	}
	next.Map[x] = s
	return next
}

type Substitution map[int]Type

func (sub Substitution) Apply(t Type) Type {
	seen := make(map[int]bool)
	return sub.applyWithSeen(t, seen)
}

func (sub Substitution) applyWithSeen(t Type, seen map[int]bool) Type {
	switch ty := t.(type) {
	case PrimType:
		return ty
	case VarType:
		if replacement, ok := sub[ty.Id]; ok {
			if seen[ty.Id] {
				return ty
			}
			seen[ty.Id] = true
			res := sub.applyWithSeen(replacement, seen)
			delete(seen, ty.Id)
			return res
		}
		return ty
	case FunType:
		return FunType{
			From: sub.applyWithSeen(ty.From, seen),
			To:   sub.applyWithSeen(ty.To, seen),
		}
	case ListType:
		return ListType{
			Elem: sub.applyWithSeen(ty.Elem, seen),
		}
	case TupleType:
		var elems []Type
		for _, e := range ty.Elems {
			elems = append(elems, sub.applyWithSeen(e, seen))
		}
		return TupleType{Elems: elems}
	case MapType:
		return MapType{
			Key: sub.applyWithSeen(ty.Key, seen),
			Val: sub.applyWithSeen(ty.Val, seen),
		}
	case SetType:
		return SetType{
			Elem: sub.applyWithSeen(ty.Elem, seen),
		}
	case VecType:
		return VecType{
			Elem: sub.applyWithSeen(ty.Elem, seen),
		}
	case PQType:
		return PQType{
			Elem: sub.applyWithSeen(ty.Elem, seen),
		}
	}
	return t
}

func (sub Substitution) ApplyScheme(s Scheme) Scheme {
	cleanSub := make(Substitution)
	bound := make(map[int]bool)
	for _, v := range s.Vars {
		bound[v] = true
	}
	for k, v := range sub {
		if !bound[k] {
			cleanSub[k] = v
		}
	}
	return Scheme{
		Vars: s.Vars,
		Ty:   cleanSub.Apply(s.Ty),
	}
}

func (sub Substitution) ApplyEnv(env *TypeEnv) *TypeEnv {
	next := NewTypeEnv(env.Parent)
	for k, v := range env.Map {
		next.Map[k] = sub.ApplyScheme(v)
	}
	return next
}

func (s1 Substitution) Compose(s2 Substitution) Substitution {
	res := make(Substitution)
	for k, v := range s2 {
		res[k] = s1.Apply(v)
	}
	for k, v := range s1 {
		if _, ok := res[k]; !ok {
			res[k] = v
		}
	}
	return res
}

func (s1 Substitution) Unify(t1, t2 Type) (Substitution, error) {
	t1 = s1.Apply(t1)
	t2 = s1.Apply(t2)

	if ty1, ok := t1.(VarType); ok {
		return s1.bind(ty1.Id, t2)
	}
	if ty2, ok := t2.(VarType); ok {
		return s1.bind(ty2.Id, t1)
	}

	switch ty1 := t1.(type) {
	case PrimType:
		if ty2, ok := t2.(PrimType); ok && ty1 == ty2 {
			return s1, nil
		}
		return nil, fmt.Errorf("cannot unify %s and %s", t1, t2)
	case FunType:
		if ty2, ok := t2.(FunType); ok {
			s2, err := s1.Unify(ty1.From, ty2.From)
			if err != nil {
				return nil, err
			}
			return s2.Unify(ty1.To, ty2.To)
		}
		return nil, fmt.Errorf("cannot unify %s and %s", t1, t2)
	case ListType:
		if ty2, ok := t2.(ListType); ok {
			return s1.Unify(ty1.Elem, ty2.Elem)
		}
		return nil, fmt.Errorf("cannot unify %s and %s", t1, t2)
	case TupleType:
		if ty2, ok := t2.(TupleType); ok {
			if len(ty1.Elems) != len(ty2.Elems) {
				return nil, fmt.Errorf("cannot unify %s and %s", t1, t2)
			}
			sCurr := s1
			var err error
			for i := range ty1.Elems {
				sCurr, err = sCurr.Unify(ty1.Elems[i], ty2.Elems[i])
				if err != nil {
					return nil, err
				}
			}
			return sCurr, nil
		}
		return nil, fmt.Errorf("cannot unify %s and %s", t1, t2)
	case MapType:
		if ty2, ok := t2.(MapType); ok {
			s2, err := s1.Unify(ty1.Key, ty2.Key)
			if err != nil {
				return nil, err
			}
			return s2.Unify(ty1.Val, ty2.Val)
		}
		return nil, fmt.Errorf("cannot unify %s and %s", t1, t2)
	case SetType:
		if ty2, ok := t2.(SetType); ok {
			return s1.Unify(ty1.Elem, ty2.Elem)
		}
		return nil, fmt.Errorf("cannot unify %s and %s", t1, t2)
	case VecType:
		if ty2, ok := t2.(VecType); ok {
			return s1.Unify(ty1.Elem, ty2.Elem)
		}
		return nil, fmt.Errorf("cannot unify %s and %s", t1, t2)
	case PQType:
		if ty2, ok := t2.(PQType); ok {
			return s1.Unify(ty1.Elem, ty2.Elem)
		}
		return nil, fmt.Errorf("cannot unify %s and %s", t1, t2)
	}

	return nil, fmt.Errorf("cannot unify %s and %s", t1, t2)
}

func (s Substitution) bind(id int, t Type) (Substitution, error) {
	if v, ok := t.(VarType); ok && v.Id == id {
		return s, nil
	}
	if occurs(id, t) {
		return nil, fmt.Errorf("occurs check failed (infinite type): %s in %s", VarType{Id: id}, t)
	}
	newSub := make(Substitution)
	newSub[id] = t
	return newSub.Compose(s), nil
}

func occurs(id int, t Type) bool {
	switch ty := t.(type) {
	case VarType:
		return ty.Id == id
	case FunType:
		return occurs(id, ty.From) || occurs(id, ty.To)
	case ListType:
		return occurs(id, ty.Elem)
	case TupleType:
		for _, e := range ty.Elems {
			if occurs(id, e) {
				return true
			}
		}
	case MapType:
		return occurs(id, ty.Key) || occurs(id, ty.Val)
	case SetType:
		return occurs(id, ty.Elem)
	case VecType:
		return occurs(id, ty.Elem)
	case PQType:
		return occurs(id, ty.Elem)
	}
	return false
}

func freeVars(t Type) map[int]bool {
	f := make(map[int]bool)
	switch ty := t.(type) {
	case VarType:
		f[ty.Id] = true
	case FunType:
		for k := range freeVars(ty.From) {
			f[k] = true
		}
		for k := range freeVars(ty.To) {
			f[k] = true
		}
	case ListType:
		for k := range freeVars(ty.Elem) {
			f[k] = true
		}
	case TupleType:
		for _, e := range ty.Elems {
			for k := range freeVars(e) {
				f[k] = true
			}
		}
	case MapType:
		for k := range freeVars(ty.Key) {
			f[k] = true
		}
		for k := range freeVars(ty.Val) {
			f[k] = true
		}
	case SetType:
		for k := range freeVars(ty.Elem) {
			f[k] = true
		}
	case VecType:
		for k := range freeVars(ty.Elem) {
			f[k] = true
		}
	case PQType:
		for k := range freeVars(ty.Elem) {
			f[k] = true
		}
	}
	return f
}

func freeVarsScheme(s Scheme) map[int]bool {
	f := freeVars(s.Ty)
	for _, v := range s.Vars {
		delete(f, v)
	}
	return f
}

func freeVarsEnv(env *TypeEnv) map[int]bool {
	f := make(map[int]bool)
	for curr := env; curr != nil; curr = curr.Parent {
		for _, s := range curr.Map {
			for k := range freeVarsScheme(s) {
				f[k] = true
			}
		}
	}
	return f
}

func Generalize(env *TypeEnv, t Type) Scheme {
	envFree := freeVarsEnv(env)
	tFree := freeVars(t)
	var bound []int
	for k := range tFree {
		if !envFree[k] {
			bound = append(bound, k)
		}
	}
	return Scheme{Vars: bound, Ty: t}
}

type TypeChecker struct {
	nextVarId int
}

func NewTypeChecker() *TypeChecker {
	return &TypeChecker{nextVarId: 1000}
}

func (tc *TypeChecker) Fresh() VarType {
	id := tc.nextVarId
	tc.nextVarId++
	return VarType{Id: id}
}

func (tc *TypeChecker) Instantiate(s Scheme) Type {
	sub := make(Substitution)
	for _, v := range s.Vars {
		sub[v] = tc.Fresh()
	}
	return sub.Apply(s.Ty)
}

type TypeError struct {
	Node ast.Node
	Err  error
}

func (e *TypeError) Unwrap() error { return e.Err }
func (e *TypeError) Error() string { return e.Err.Error() }

func (tc *TypeChecker) Infer(env *TypeEnv, node ast.Node, sub Substitution) (Type, Substitution, error) {
	t, s, err := tc.inferInternal(env, node, sub)
	if err != nil {
		if _, ok := err.(*TypeError); ok {
			return nil, nil, err
		}
		return nil, nil, &TypeError{Node: node, Err: err}
	}
	return t, s, nil
}

func (tc *TypeChecker) inferInternal(env *TypeEnv, node ast.Node, sub Substitution) (Type, Substitution, error) {
	switch n := node.(type) {
	case ast.IntNode:
		return TInt, sub, nil
	case ast.RealNode:
		// Miranda unifies integers and reals under one type `num` (TInt here);
		// the int/real distinction is purely a runtime matter.
		return TInt, sub, nil
	case ast.BoolNode:
		return TBool, sub, nil
	case ast.CharNode:
		return TChar, sub, nil
	case ast.NilNode:
		elem := tc.Fresh()
		return ListType{Elem: elem}, sub, nil

	case ast.ConsNode:
		tH, sub1, err := tc.Infer(env, n.Head, sub)
		if err != nil {
			return nil, nil, err
		}
		tT, sub2, err := tc.Infer(env, n.Tail, sub1)
		if err != nil {
			return nil, nil, err
		}
		sub3, err := sub2.Unify(tT, ListType{Elem: tH})
		if err != nil {
			return nil, nil, fmt.Errorf("list cons type error: %w", err)
		}
		return sub3.Apply(tT), sub3, nil

	case ast.TupleNode:
		var elems []Type
		sCurr := sub
		for _, e := range n.Elems {
			tE, sNext, err := tc.Infer(env, e, sCurr)
			if err != nil {
				return nil, nil, err
			}
			elems = append(elems, tE)
			sCurr = sNext
		}
		for i := range elems {
			elems[i] = sCurr.Apply(elems[i])
		}
		return TupleType{Elems: elems}, sCurr, nil

	case ast.VarNode:
		s, ok := env.Lookup(n.Name)
		if !ok {
			return nil, nil, fmt.Errorf("unbound variable: %s", n.Name)
		}
		instTy := tc.Instantiate(s)
		return sub.Apply(instTy), sub, nil

	case ast.LamNode:
		paramTy := tc.Fresh()
		extendedEnv := env.Extend(n.Var, Scheme{Vars: nil, Ty: paramTy})
		bodyTy, sub1, err := tc.Infer(extendedEnv, n.Body, sub)
		if err != nil {
			return nil, nil, err
		}
		resTy := FunType{
			From: sub1.Apply(paramTy),
			To:   bodyTy,
		}
		return resTy, sub1, nil

	case ast.AppNode:
		tL, sub1, err := tc.Infer(env, n.Left, sub)
		if err != nil {
			return nil, nil, err
		}
		tR, sub2, err := tc.Infer(env, n.Right, sub1)
		if err != nil {
			return nil, nil, err
		}
		resTy := tc.Fresh()
		sub3, err := sub2.Unify(tL, FunType{From: tR, To: resTy})
		if err != nil {
			return nil, nil, fmt.Errorf("function application type error: %w", err)
		}
		return sub3.Apply(resTy), sub3, nil

	case ast.LetNode:
		envPrime := env
		var bindingsTypes []Type
		var bindingsNames []string
		sCurr := sub

		for _, b := range n.Bindings {
			selfTy := tc.Fresh()
			envPrime = envPrime.Extend(b.Name, Scheme{Vars: nil, Ty: selfTy})
			bindingsTypes = append(bindingsTypes, selfTy)
			bindingsNames = append(bindingsNames, b.Name)
		}

		for idx, b := range n.Bindings {
			tB, sNext, err := tc.Infer(envPrime, b.Expr, sCurr)
			if err != nil {
				return nil, nil, err
			}
			sNext2, err := sNext.Unify(bindingsTypes[idx], tB)
			if err != nil {
				return nil, nil, fmt.Errorf("type error in local binding '%s': %w", b.Name, err)
			}
			sCurr = sNext2
		}

		envGeneralized := env
		for idx, name := range bindingsNames {
			finalTy := sCurr.Apply(bindingsTypes[idx])
			scheme := Generalize(sCurr.ApplyEnv(envGeneralized), finalTy)
			envGeneralized = envGeneralized.Extend(name, scheme)
		}

		return tc.Infer(envGeneralized, n.Body, sCurr)

	case ast.IfNode:
		tCond, sub1, err := tc.Infer(env, n.Cond, sub)
		if err != nil {
			return nil, nil, err
		}
		sub2, err := sub1.Unify(tCond, TBool)
		if err != nil {
			return nil, nil, fmt.Errorf("if condition must be Bool: %w", err)
		}
		tThen, sub3, err := tc.Infer(env, n.Then, sub2)
		if err != nil {
			return nil, nil, err
		}
		tElse, sub4, err := tc.Infer(env, n.Else, sub3)
		if err != nil {
			return nil, nil, err
		}
		sub5, err := sub4.Unify(tThen, tElse)
		if err != nil {
			return nil, nil, fmt.Errorf("then/else branches must have the same type: %w", err)
		}
		return sub5.Apply(tThen), sub5, nil

	case ast.IfZeroNode:
		tCond, sub1, err := tc.Infer(env, n.Cond, sub)
		if err != nil {
			return nil, nil, err
		}
		sub2, err := sub1.Unify(tCond, TInt)
		if err != nil {
			return nil, nil, fmt.Errorf("ifzero condition must be Int: %w", err)
		}
		tThen, sub3, err := tc.Infer(env, n.Then, sub2)
		if err != nil {
			return nil, nil, err
		}
		tElse, sub4, err := tc.Infer(env, n.Else, sub3)
		if err != nil {
			return nil, nil, err
		}
		sub5, err := sub4.Unify(tThen, tElse)
		if err != nil {
			return nil, nil, fmt.Errorf("then/else branches must have the same type: %w", err)
		}
		return sub5.Apply(tThen), sub5, nil

	case ast.IfNilNode:
		tCond, sub1, err := tc.Infer(env, n.Cond, sub)
		if err != nil {
			return nil, nil, err
		}
		elem := tc.Fresh()
		sub2, err := sub1.Unify(tCond, ListType{Elem: elem})
		if err != nil {
			return nil, nil, fmt.Errorf("ifnil condition must be a List: %w", err)
		}
		tThen, sub3, err := tc.Infer(env, n.Then, sub2)
		if err != nil {
			return nil, nil, err
		}
		tElse, sub4, err := tc.Infer(env, n.Else, sub3)
		if err != nil {
			return nil, nil, err
		}
		sub5, err := sub4.Unify(tThen, tElse)
		if err != nil {
			return nil, nil, fmt.Errorf("then/else branches must have the same type: %w", err)
		}
		return sub5.Apply(tThen), sub5, nil

	case ast.ProjNode:
		tT, sub1, err := tc.Infer(env, n.Tuple, sub)
		if err != nil {
			return nil, nil, err
		}
		tTuple, ok := sub1.Apply(tT).(TupleType)
		if !ok {
			var freshElems []Type
			for i := 0; i <= n.Index; i++ {
				freshElems = append(freshElems, tc.Fresh())
			}
			tTupleObj := TupleType{Elems: freshElems}
			sub2, err := sub1.Unify(tT, tTupleObj)
			if err != nil {
				return nil, nil, fmt.Errorf("proj expects a tuple: %w", err)
			}
			tTuple = sub2.Apply(tTupleObj).(TupleType)
			sub1 = sub2
		}
		if n.Index < 0 || n.Index >= len(tTuple.Elems) {
			return nil, nil, fmt.Errorf("projection index %d out of bounds for tuple %s", n.Index, tTuple)
		}
		return sub1.Apply(tTuple.Elems[n.Index]), sub1, nil

	case ast.IndexNode:
		tList, sub1, err := tc.Infer(env, n.List, sub)
		if err != nil {
			return nil, nil, err
		}
		tIdx, sub2, err := tc.Infer(env, n.Index, sub1)
		if err != nil {
			return nil, nil, err
		}
		sub3, err := sub2.Unify(tIdx, TInt)
		if err != nil {
			return nil, nil, fmt.Errorf("'!' index must be Int: %w", err)
		}
		elem := tc.Fresh()
		sub4, err := sub3.Unify(tList, ListType{Elem: elem})
		if err != nil {
			return nil, nil, fmt.Errorf("'!' expects a list: %w", err)
		}
		return sub4.Apply(elem), sub4, nil

	case ast.AddNode, ast.SubNode, ast.MulNode, ast.DivNode, ast.IDivNode, ast.ModNode, ast.PowNode:
		var leftNode, rightNode ast.Node
		var opName string
		switch op := n.(type) {
		case ast.AddNode:
			leftNode, rightNode, opName = op.Left, op.Right, "+"
		case ast.SubNode:
			leftNode, rightNode, opName = op.Left, op.Right, "-"
		case ast.MulNode:
			leftNode, rightNode, opName = op.Left, op.Right, "*"
		case ast.DivNode:
			leftNode, rightNode, opName = op.Left, op.Right, "/"
		case ast.IDivNode:
			leftNode, rightNode, opName = op.Left, op.Right, "div"
		case ast.ModNode:
			leftNode, rightNode, opName = op.Left, op.Right, "mod"
		case ast.PowNode:
			leftNode, rightNode, opName = op.Left, op.Right, "^"
		}
		tL, sub1, err := tc.Infer(env, leftNode, sub)
		if err != nil {
			return nil, nil, err
		}
		tR, sub2, err := tc.Infer(env, rightNode, sub1)
		if err != nil {
			return nil, nil, err
		}
		sub3, err := sub2.Unify(tL, TInt)
		if err != nil {
			return nil, nil, fmt.Errorf("operator '%s' expects Int: %w", opName, err)
		}
		sub4, err := sub3.Unify(tR, TInt)
		if err != nil {
			return nil, nil, fmt.Errorf("operator '%s' expects Int: %w", opName, err)
		}
		return TInt, sub4, nil

	case ast.EqNode, ast.NeNode:
		var leftNode, rightNode ast.Node
		var opName string
		switch op := n.(type) {
		case ast.EqNode:
			leftNode, rightNode, opName = op.Left, op.Right, "=="
		case ast.NeNode:
			leftNode, rightNode, opName = op.Left, op.Right, "~="
		}
		tL, sub1, err := tc.Infer(env, leftNode, sub)
		if err != nil {
			return nil, nil, err
		}
		tR, sub2, err := tc.Infer(env, rightNode, sub1)
		if err != nil {
			return nil, nil, err
		}
		sub3, err := sub2.Unify(tL, tR)
		if err != nil {
			return nil, nil, fmt.Errorf("operator '%s' type error: cannot compare %s and %s", opName, tL, tR)
		}
		return TBool, sub3, nil

	case ast.LtNode, ast.GtNode, ast.LeNode, ast.GeNode:
		var leftNode, rightNode ast.Node
		var opName string
		switch op := n.(type) {
		case ast.LtNode:
			leftNode, rightNode, opName = op.Left, op.Right, "<"
		case ast.GtNode:
			leftNode, rightNode, opName = op.Left, op.Right, ">"
		case ast.LeNode:
			leftNode, rightNode, opName = op.Left, op.Right, "<="
		case ast.GeNode:
			leftNode, rightNode, opName = op.Left, op.Right, ">="
		}
		tL, sub1, err := tc.Infer(env, leftNode, sub)
		if err != nil {
			return nil, nil, err
		}
		tR, sub2, err := tc.Infer(env, rightNode, sub1)
		if err != nil {
			return nil, nil, err
		}
		// Ordering is polymorphic and structural, like == : the two operands
		// must have the same type (num, char, bool, or lists/tuples thereof),
		// and the evaluator compares them lexicographically.
		sub3, err := sub2.Unify(tL, tR)
		if err != nil {
			return nil, nil, fmt.Errorf("operator '%s' type error: cannot compare %s and %s", opName, tL, tR)
		}
		return TBool, sub3, nil

	case ast.AppendNode, ast.DiffNode:
		var leftNode, rightNode ast.Node
		var opName string
		switch op := n.(type) {
		case ast.AppendNode:
			leftNode, rightNode, opName = op.Left, op.Right, "++"
		case ast.DiffNode:
			leftNode, rightNode, opName = op.Left, op.Right, "--"
		}
		tL, sub1, err := tc.Infer(env, leftNode, sub)
		if err != nil {
			return nil, nil, err
		}
		tR, sub2, err := tc.Infer(env, rightNode, sub1)
		if err != nil {
			return nil, nil, err
		}
		elem := tc.Fresh()
		sub3, err := sub2.Unify(tL, ListType{Elem: elem})
		if err != nil {
			return nil, nil, fmt.Errorf("operator '%s' expects List: %w", opName, err)
		}
		sub4, err := sub3.Unify(tR, tL)
		if err != nil {
			return nil, nil, fmt.Errorf("operator '%s' expects matching List types: %w", opName, err)
		}
		return sub4.Apply(tL), sub4, nil

	case ast.RangeNode:
		tS, sub1, err := tc.Infer(env, n.Start, sub)
		if err != nil {
			return nil, nil, err
		}
		tE, sub2, err := tc.Infer(env, n.End, sub1)
		if err != nil {
			return nil, nil, err
		}
		sub3, err := sub2.Unify(tS, TInt)
		if err != nil {
			return nil, nil, fmt.Errorf("range start must be Int: %w", err)
		}
		sub4, err := sub3.Unify(tE, TInt)
		if err != nil {
			return nil, nil, fmt.Errorf("range end must be Int: %w", err)
		}
		return ListType{Elem: TInt}, sub4, nil

	case ast.RangeFromNode:
		tS, sub1, err := tc.Infer(env, n.Start, sub)
		if err != nil {
			return nil, nil, err
		}
		sub2, err := sub1.Unify(tS, TInt)
		if err != nil {
			return nil, nil, fmt.Errorf("range start must be Int: %w", err)
		}
		return ListType{Elem: TInt}, sub2, nil

	case ast.RangeStepNode:
		s := sub
		for _, e := range []ast.Node{n.Start, n.Step, n.End} {
			tE, s1, err := tc.Infer(env, e, s)
			if err != nil {
				return nil, nil, err
			}
			s, err = s1.Unify(tE, TInt)
			if err != nil {
				return nil, nil, fmt.Errorf("stepped range bounds must be Int: %w", err)
			}
		}
		return ListType{Elem: TInt}, s, nil

	case ast.RangeStepFromNode:
		s := sub
		for _, e := range []ast.Node{n.Start, n.Step} {
			tE, s1, err := tc.Infer(env, e, s)
			if err != nil {
				return nil, nil, err
			}
			s, err = s1.Unify(tE, TInt)
			if err != nil {
				return nil, nil, fmt.Errorf("stepped range bounds must be Int: %w", err)
			}
		}
		return ListType{Elem: TInt}, s, nil

	case ast.ZFNode:
		envCurr := env
		sCurr := sub
		var err error
		for _, q := range n.Quals {
			switch qual := q.(type) {
			case ast.GeneratorQual:
				tSrc, sNext, err := tc.Infer(envCurr, qual.Src, sCurr)
				if err != nil {
					return nil, nil, err
				}
				elem := tc.Fresh()
				sNext2, err := sNext.Unify(tSrc, ListType{Elem: elem})
				if err != nil {
					return nil, nil, fmt.Errorf("comprehension generator source must be a List: %w", err)
				}
				sCurr = sNext2
				envCurr, err = tc.bindPat(envCurr, qual.Pat, sCurr.Apply(elem))
				if err != nil {
					return nil, nil, err
				}
			case ast.FilterQual:
				tCond, sNext, err := tc.Infer(envCurr, qual.Cond, sCurr)
				if err != nil {
					return nil, nil, err
				}
				sNext2, err := sNext.Unify(tCond, TBool)
				if err != nil {
					return nil, nil, fmt.Errorf("comprehension filter must be Bool: %w", err)
				}
				sCurr = sNext2
			}
		}
		tBody, sFinal, err := tc.Infer(envCurr, n.Body, sCurr)
		if err != nil {
			return nil, nil, err
		}
		return ListType{Elem: sFinal.Apply(tBody)}, sFinal, nil

	case ast.ZFGeneratorNode:
		return nil, nil, fmt.Errorf("unsupported direct ZFGeneratorNode type check")
	case ast.MatchErrorNode:
		return tc.Fresh(), sub, nil
	case ast.ThunkNode:
		if n.Cell != nil && n.Cell.State == ast.Evaluated {
			return tc.Infer(env, n.Cell.Val, sub)
		}
		// If it's a thunk, we can't type check its inner expression easily without potentially
		// causing issue, but we can type check the thunk's Expr field if present.
		// Wait, a ThunkNode always has a Cell or can be evaluated. If n.Cell is nil, we can look at other fields,
		// but standard ThunkNode has Cell. If n.Cell has state Evaluated, we infer it. Otherwise, we can return a fresh variable or infer the cell's initial value expression.
		// Actually, ThunkNode has no other AST Node field except Val inside the Cell when evaluated.
		// Wait! During type checking before execution, we don't have ThunkNodes in the AST at all!
		// They are only created at runtime during Whnf evaluation. So type checking the static AST parsed from source will never encounter a ThunkNode or ClosureNode.
		// But in case any runtime evaluation mixes with type inference, returning a fresh variable is safe.
		return tc.Fresh(), sub, nil
	case ast.ClosureNode:
		// Similar to LamNode: Var is param, Body is body, Env is closure environment.
		paramTy := tc.Fresh()
		extendedEnv := env.Extend(n.Var, Scheme{Vars: nil, Ty: paramTy})
		bodyTy, sub1, err := tc.Infer(extendedEnv, n.Body, sub)
		if err != nil {
			return nil, nil, err
		}
		resTy := FunType{
			From: sub1.Apply(paramTy),
			To:   bodyTy,
		}
		return resTy, sub1, nil
	}

	return nil, nil, fmt.Errorf("unknown AST node type: %T", node)
}

func (tc *TypeChecker) bindPat(env *TypeEnv, pat ast.Pat, t Type) (*TypeEnv, error) {
	switch p := pat.(type) {
	case ast.PatVar:
		if p.Name == "_" {
			return env, nil
		}
		return env.Extend(p.Name, Scheme{Vars: nil, Ty: t}), nil
	case ast.PatInt:
		return env, nil
	case ast.PatBool:
		return env, nil
	case ast.PatChar:
		return env, nil
	case ast.PatNil:
		return env, nil
	case ast.PatCons:
		var elem Type
		if lt, ok := t.(ListType); ok {
			elem = lt.Elem
		} else {
			elem = tc.Fresh()
		}
		envH, err := tc.bindPat(env, p.Head, elem)
		if err != nil {
			return nil, err
		}
		return tc.bindPat(envH, p.Tail, ListType{Elem: elem})
	case ast.PatTuple:
		var elems []Type
		if tt, ok := t.(TupleType); ok && len(tt.Elems) == len(p.Elems) {
			elems = tt.Elems
		} else {
			for i := 0; i < len(p.Elems); i++ {
				elems = append(elems, tc.Fresh())
			}
		}
		envCurr := env
		var err error
		for i := range p.Elems {
			envCurr, err = tc.bindPat(envCurr, p.Elems[i], elems[i])
			if err != nil {
				return nil, err
			}
		}
		return envCurr, nil
	}
	return env, nil
}

func DefaultTypeEnv() *TypeEnv {
	env := NewTypeEnv(nil)

	env.Map["True"] = Scheme{Vars: nil, Ty: TBool}
	env.Map["False"] = Scheme{Vars: nil, Ty: TBool}

	env.Map["hd"] = Scheme{Vars: []int{0}, Ty: FunType{From: ListType{Elem: VarType{Id: 0}}, To: VarType{Id: 0}}}
	env.Map["tl"] = Scheme{Vars: []int{0}, Ty: FunType{From: ListType{Elem: VarType{Id: 0}}, To: ListType{Elem: VarType{Id: 0}}}}
	env.Map["show"] = Scheme{Vars: []int{0}, Ty: FunType{From: VarType{Id: 0}, To: ListType{Elem: TChar}}}
	env.Map["read"] = Scheme{Vars: []int{0}, Ty: FunType{From: ListType{Elem: TChar}, To: VarType{Id: 0}}}
	env.Map["lines"] = Scheme{Vars: nil, Ty: FunType{From: ListType{Elem: TChar}, To: ListType{Elem: ListType{Elem: TChar}}}}
	env.Map["numval"] = Scheme{Vars: nil, Ty: FunType{From: ListType{Elem: TChar}, To: TInt}}
	env.Map["length"] = Scheme{Vars: []int{0}, Ty: FunType{From: ListType{Elem: VarType{Id: 0}}, To: TInt}}
	env.Map["seq"] = Scheme{Vars: []int{0, 1}, Ty: FunType{From: VarType{Id: 0}, To: FunType{From: VarType{Id: 1}, To: VarType{Id: 1}}}}
	env.Map["reverse"] = Scheme{Vars: []int{0}, Ty: FunType{From: ListType{Elem: VarType{Id: 0}}, To: ListType{Elem: VarType{Id: 0}}}}

	env.Map["h_lookup"] = Scheme{Vars: []int{0, 1}, Ty: FunType{From: MapType{Key: VarType{Id: 0}, Val: VarType{Id: 1}}, To: FunType{From: VarType{Id: 0}, To: VarType{Id: 1}}}}
	env.Map["h_insert"] = Scheme{Vars: []int{0, 1}, Ty: FunType{From: MapType{Key: VarType{Id: 0}, Val: VarType{Id: 1}}, To: FunType{From: VarType{Id: 0}, To: FunType{From: VarType{Id: 1}, To: MapType{Key: VarType{Id: 0}, Val: VarType{Id: 1}}}}}}
	env.Map["member"] = Scheme{Vars: []int{0}, Ty: FunType{From: SetType{Elem: VarType{Id: 0}}, To: FunType{From: VarType{Id: 0}, To: TBool}}}
	env.Map["s_insert"] = Scheme{Vars: []int{0}, Ty: FunType{From: SetType{Elem: VarType{Id: 0}}, To: FunType{From: VarType{Id: 0}, To: SetType{Elem: VarType{Id: 0}}}}}
	env.Map["split"] = Scheme{Vars: nil, Ty: FunType{From: ListType{Elem: TChar}, To: FunType{From: ListType{Elem: TChar}, To: ListType{Elem: ListType{Elem: TChar}}}}}
	env.Map["parse_ints"] = Scheme{Vars: nil, Ty: FunType{From: ListType{Elem: TChar}, To: ListType{Elem: TInt}}}
	env.Map["list_get"] = Scheme{Vars: nil, Ty: FunType{From: ListType{Elem: TInt}, To: FunType{From: TInt, To: TInt}}}
	env.Map["list_set"] = Scheme{Vars: nil, Ty: FunType{From: ListType{Elem: TInt}, To: FunType{From: TInt, To: FunType{From: TInt, To: ListType{Elem: TInt}}}}}
	env.Map["memoize"] = Scheme{Vars: []int{0, 1}, Ty: FunType{From: FunType{From: VarType{Id: 0}, To: VarType{Id: 1}}, To: FunType{From: VarType{Id: 0}, To: VarType{Id: 1}}}}
	env.Map["sort_by"] = Scheme{Vars: []int{0}, Ty: FunType{From: FunType{From: VarType{Id: 0}, To: FunType{From: VarType{Id: 0}, To: TInt}}, To: FunType{From: ListType{Elem: VarType{Id: 0}}, To: ListType{Elem: VarType{Id: 0}}}}}
	env.Map["empty_map"] = Scheme{Vars: []int{0, 1}, Ty: MapType{Key: VarType{Id: 0}, Val: VarType{Id: 1}}}
	env.Map["empty_set"] = Scheme{Vars: []int{0}, Ty: SetType{Elem: VarType{Id: 0}}}
	env.Map["sort_ints"] = Scheme{Vars: nil, Ty: FunType{From: ListType{Elem: TInt}, To: ListType{Elem: TInt}}}
	env.Map["sort_edges"] = Scheme{Vars: nil, Ty: FunType{From: ListType{Elem: TupleType{Elems: []Type{TInt, TInt, TInt}}}, To: ListType{Elem: TupleType{Elems: []Type{TInt, TInt, TInt}}}}}
	env.Map["sort_pts"] = Scheme{Vars: nil, Ty: FunType{From: ListType{Elem: TupleType{Elems: []Type{TInt, TupleType{Elems: []Type{TInt, TInt, TInt}}}}}, To: ListType{Elem: TupleType{Elems: []Type{TInt, TupleType{Elems: []Type{TInt, TInt, TInt}}}}}}}
	env.Map["h_lookup_def"] = Scheme{Vars: []int{0, 1}, Ty: FunType{From: MapType{Key: VarType{Id: 0}, Val: VarType{Id: 1}}, To: FunType{From: VarType{Id: 0}, To: FunType{From: VarType{Id: 1}, To: VarType{Id: 1}}}}}
	env.Map["to_vec"] = Scheme{Vars: []int{0}, Ty: FunType{From: ListType{Elem: VarType{Id: 0}}, To: VecType{Elem: VarType{Id: 0}}}}
	env.Map["vec_get"] = Scheme{Vars: []int{0}, Ty: FunType{From: VecType{Elem: VarType{Id: 0}}, To: FunType{From: TInt, To: VarType{Id: 0}}}}
	env.Map["vec_len"] = Scheme{Vars: []int{0}, Ty: FunType{From: VecType{Elem: VarType{Id: 0}}, To: TInt}}
	env.Map["vec_set"] = Scheme{Vars: []int{0}, Ty: FunType{From: VecType{Elem: VarType{Id: 0}}, To: FunType{From: TInt, To: FunType{From: VarType{Id: 0}, To: VecType{Elem: VarType{Id: 0}}}}}}
	env.Map["vec_to_list"] = Scheme{Vars: []int{0}, Ty: FunType{From: VecType{Elem: VarType{Id: 0}}, To: ListType{Elem: VarType{Id: 0}}}}

	// bitwise operators on integers: num -> num -> num
	numNumNum := FunType{From: TInt, To: FunType{From: TInt, To: TInt}}
	env.Map["ord"] = Scheme{Vars: nil, Ty: FunType{From: TChar, To: TInt}}
	env.Map["chr"] = Scheme{Vars: nil, Ty: FunType{From: TInt, To: TChar}}
	// num -> num transcendental / real functions; entier floors a real to an int
	numNum := FunType{From: TInt, To: TInt}
	for _, fn := range []string{"sqrt", "sin", "cos", "tan", "atan", "exp", "log", "entier"} {
		env.Map[fn] = Scheme{Vars: nil, Ty: numNum}
	}
	env.Map["xor"] = Scheme{Vars: nil, Ty: numNumNum}
	env.Map["band"] = Scheme{Vars: nil, Ty: numNumNum}
	env.Map["bor"] = Scheme{Vars: nil, Ty: numNumNum}
	env.Map["shl"] = Scheme{Vars: nil, Ty: numNumNum}
	env.Map["shr"] = Scheme{Vars: nil, Ty: numNumNum}

	// memofix :: ((a -> b) -> a -> b) -> a -> b  (memoized open recursion)
	aToB := FunType{From: VarType{Id: 0}, To: VarType{Id: 1}}
	env.Map["memofix"] = Scheme{Vars: []int{0, 1}, Ty: FunType{From: FunType{From: aToB, To: aToB}, To: aToB}}

	// priority queue (min-heap keyed by an integer priority)
	pqA := PQType{Elem: VarType{Id: 0}}
	env.Map["pq_empty"] = Scheme{Vars: []int{0}, Ty: pqA}
	env.Map["pq_push"] = Scheme{Vars: []int{0}, Ty: FunType{From: pqA, To: FunType{From: TInt, To: FunType{From: VarType{Id: 0}, To: pqA}}}}
	env.Map["pq_pop"] = Scheme{Vars: []int{0}, Ty: FunType{From: pqA, To: TupleType{Elems: []Type{TInt, VarType{Id: 0}, pqA}}}}
	env.Map["pq_null"] = Scheme{Vars: []int{0}, Ty: FunType{From: pqA, To: TBool}}

	return env
}
