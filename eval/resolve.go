package eval

import "pkreyenhop.com/miracula-go/ast"

// resolverBuiltins mirrors the names the evaluator dispatches natively in
// its VarNode case. The evaluator checks them before any environment
// lookup, so they shadow locals and globals and must stay VarNodes.
var resolverBuiltins = map[string]bool{
	"hd": true, "tl": true, "show": true, "read": true, "lines": true,
	"numval": true, "length": true, "reverse": true, "seq": true,
	"h_lookup": true, "h_insert": true, "member": true, "split": true,
	"parse_ints": true, "list_get": true, "list_set": true, "memoize": true,
	"sort_by": true, "sort_ints": true, "sort_edges": true, "sort_pts": true,
	"h_lookup_def": true, "empty_map": true, "empty_set": true,
}

// Resolve rewrites variable references in a freshly desugared definition or
// REPL expression into lexical coordinates. A reference to an enclosing
// LamNode, LetNode, or comprehension-generator binding becomes
// LocalVarNode{Depth} — frames are single-binding, so the evaluator reaches
// the value with Depth pointer hops and no name comparison — and any other
// reference becomes GlobalVarNode, read straight from the Globals map.
// Runtime-only nodes (thunks, closures, partial applications) never appear
// in freshly parsed code and pass through untouched, as does any VarNode
// naming a builtin.
func Resolve(n ast.Node) ast.Node {
	return resolveNode(n, nil)
}

// pushScope returns a fresh scope slice with the given names appended; the
// copy keeps sibling branches from sharing (and clobbering) backing arrays.
// scope is the static mirror of the run-time environment chain: one name
// per single-binding frame, innermost last.
func pushScope(scope []string, names ...string) []string {
	out := make([]string, 0, len(scope)+len(names))
	out = append(out, scope...)
	return append(out, names...)
}

// patVars lists the variables a pattern binds, in the order matchPattern
// and mergeBindings produce them at run time: traversal order, wildcards
// excluded, duplicates keeping their first position.
func patVars(p ast.Pat) []string {
	var out []string
	seen := make(map[string]bool)
	var walk func(ast.Pat)
	walk = func(p ast.Pat) {
		switch pt := p.(type) {
		case ast.PatVar:
			if pt.Name != "_" && !seen[pt.Name] {
				seen[pt.Name] = true
				out = append(out, pt.Name)
			}
		case ast.PatCons:
			walk(pt.Head)
			walk(pt.Tail)
		case ast.PatTuple:
			for _, e := range pt.Elems {
				walk(e)
			}
		}
	}
	walk(p)
	return out
}

func resolveNode(n ast.Node, scope []string) ast.Node {
	switch node := n.(type) {
	case ast.VarNode:
		if resolverBuiltins[node.Name] {
			return node
		}
		for i := len(scope) - 1; i >= 0; i-- {
			if scope[i] == node.Name {
				return ast.LocalVarNode{Depth: len(scope) - 1 - i, Name: node.Name}
			}
		}
		return ast.GlobalVarNode{Name: node.Name}
	case ast.LamNode:
		return ast.LamNode{Var: node.Var, Body: resolveNode(node.Body, pushScope(scope, node.Var))}
	case ast.LetNode:
		// letrec: every binding expression and the body see all bindings,
		// one frame per binding in declaration order (innermost = last),
		// exactly as the evaluator extends the environment
		names := make([]string, len(node.Bindings))
		for i, b := range node.Bindings {
			names[i] = b.Name
		}
		inner := pushScope(scope, names...)
		bindings := make([]ast.Binding, len(node.Bindings))
		for i, b := range node.Bindings {
			bindings[i] = ast.Binding{Name: b.Name, Expr: resolveNode(b.Expr, inner)}
		}
		return ast.LetNode{Bindings: bindings, Body: resolveNode(node.Body, inner)}
	case ast.ZFNode:
		// qualifiers scope left to right: a generator's source sees the
		// frames of the generators before it, and its pattern variables
		// (in patVars order) are in scope for everything after it
		cur := scope
		quals := make([]ast.Qualifier, len(node.Quals))
		for i, q := range node.Quals {
			switch qual := q.(type) {
			case ast.FilterQual:
				quals[i] = ast.FilterQual{Cond: resolveNode(qual.Cond, cur)}
			case ast.GeneratorQual:
				src := resolveNode(qual.Src, cur)
				quals[i] = ast.GeneratorQual{Pat: qual.Pat, Src: src}
				cur = pushScope(cur, patVars(qual.Pat)...)
			default:
				quals[i] = q
			}
		}
		return ast.ZFNode{Body: resolveNode(node.Body, cur), Quals: quals}
	case ast.AppNode:
		return ast.AppNode{Left: resolveNode(node.Left, scope), Right: resolveNode(node.Right, scope)}
	case ast.ConsNode:
		return ast.ConsNode{Head: resolveNode(node.Head, scope), Tail: resolveNode(node.Tail, scope)}
	case ast.TupleNode:
		elems := make([]ast.Node, len(node.Elems))
		for i, e := range node.Elems {
			elems[i] = resolveNode(e, scope)
		}
		return ast.TupleNode{Elems: elems}
	case ast.ProjNode:
		return ast.ProjNode{Index: node.Index, Tuple: resolveNode(node.Tuple, scope)}
	case ast.IfNode:
		return ast.IfNode{Cond: resolveNode(node.Cond, scope), Then: resolveNode(node.Then, scope), Else: resolveNode(node.Else, scope)}
	case ast.IfZeroNode:
		return ast.IfZeroNode{Cond: resolveNode(node.Cond, scope), Then: resolveNode(node.Then, scope), Else: resolveNode(node.Else, scope)}
	case ast.IfNilNode:
		return ast.IfNilNode{Cond: resolveNode(node.Cond, scope), Then: resolveNode(node.Then, scope), Else: resolveNode(node.Else, scope)}
	case ast.AppendNode:
		return ast.AppendNode{Left: resolveNode(node.Left, scope), Right: resolveNode(node.Right, scope)}
	case ast.DiffNode:
		return ast.DiffNode{Left: resolveNode(node.Left, scope), Right: resolveNode(node.Right, scope)}
	case ast.RangeNode:
		return ast.RangeNode{Start: resolveNode(node.Start, scope), End: resolveNode(node.End, scope)}
	case ast.AddNode:
		return ast.AddNode{Left: resolveNode(node.Left, scope), Right: resolveNode(node.Right, scope)}
	case ast.SubNode:
		return ast.SubNode{Left: resolveNode(node.Left, scope), Right: resolveNode(node.Right, scope)}
	case ast.MulNode:
		return ast.MulNode{Left: resolveNode(node.Left, scope), Right: resolveNode(node.Right, scope)}
	case ast.DivNode:
		return ast.DivNode{Left: resolveNode(node.Left, scope), Right: resolveNode(node.Right, scope)}
	case ast.ModNode:
		return ast.ModNode{Left: resolveNode(node.Left, scope), Right: resolveNode(node.Right, scope)}
	case ast.EqNode:
		return ast.EqNode{Left: resolveNode(node.Left, scope), Right: resolveNode(node.Right, scope)}
	case ast.NeNode:
		return ast.NeNode{Left: resolveNode(node.Left, scope), Right: resolveNode(node.Right, scope)}
	case ast.LtNode:
		return ast.LtNode{Left: resolveNode(node.Left, scope), Right: resolveNode(node.Right, scope)}
	case ast.GtNode:
		return ast.GtNode{Left: resolveNode(node.Left, scope), Right: resolveNode(node.Right, scope)}
	case ast.LeNode:
		return ast.LeNode{Left: resolveNode(node.Left, scope), Right: resolveNode(node.Right, scope)}
	case ast.GeNode:
		return ast.GeNode{Left: resolveNode(node.Left, scope), Right: resolveNode(node.Right, scope)}
	default:
		// literals (Int/Bool/Char/Nil/MatchError) and runtime-only nodes
		return n
	}
}
