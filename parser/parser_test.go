package parser

import (
	"strings"
	"testing"
	"pkreyenhop.com/miracula-go/ast"
	"pkreyenhop.com/miracula-go/lexer"
)

func TestParseExpr(t *testing.T) {
	toks := lexer.Tokenize("1 + 2 * 3")
	p := NewParser(toks)
	stmt := p.Parse()

	evalStmt, ok := stmt.(REPLEvalStmt)
	if !ok {
		t.Fatalf("Expected REPLEvalStmt, got %T", stmt)
	}

	// Verify precedence: 1 + (2 * 3)
	addNode, ok := evalStmt.Expr.(ast.AddNode)
	if !ok {
		t.Fatalf("Expected AddNode at root, got %T", evalStmt.Expr)
	}

	leftInt, ok := addNode.Left.(ast.IntNode)
	if !ok || leftInt.Val != 1 {
		t.Errorf("Expected Left to be IntNode(1), got %T", addNode.Left)
	}

	_, ok = addNode.Right.(ast.MulNode)
	if !ok {
		t.Errorf("Expected Right to be MulNode, got %T", addNode.Right)
	}
}

func TestDesugarEquations(t *testing.T) {
	// Let's desugar:
	// f 0 = 1
	// f x = x
	eqs := []RawBinding{
		{
			FName: "f",
			Pats:  []ast.Pat{ast.PatInt{Val: 0}},
			Body:  ast.IntNode{Val: 1},
		},
		{
			FName: "f",
			Pats:  []ast.Pat{ast.PatVar{Name: "x"}},
			Body:  ast.VarNode{Name: "x"},
		},
	}

	node := DesugarEquations(eqs)
	if node == nil {
		t.Fatalf("Desugaring returned nil")
	}

	// We expect a lambda node since f has arity 1.
	lam, ok := node.(ast.LamNode)
	if !ok {
		t.Fatalf("Expected LamNode at root of desugared equation, got %T", node)
	}

	// synthetic parameter names are '$'-prefixed so they cannot collide with
	// user identifiers (which may never contain '$')
	if !strings.HasPrefix(lam.Var, "$p") {
		t.Errorf("Expected a synthetic '$p' parameter variable, got %s", lam.Var)
	}
}

func TestParseGuards(t *testing.T) {
	toks := lexer.Tokenize("abs x = -x, if x < 0 = x, otherwise")
	p := NewParser(toks)
	stmt := p.Parse()

	bind, ok := stmt.(ScriptBindStmt)
	if !ok {
		t.Fatalf("Expected ScriptBindStmt, got %T", stmt)
	}

	if bind.Binding.FName != "abs" {
		t.Errorf("Expected FName 'abs', got %s", bind.Binding.FName)
	}

	// Body should be an IfNode (desugared guard)
	ifNode, ok := bind.Binding.Body.(ast.IfNode)
	if !ok {
		t.Fatalf("Expected IfNode at body root, got %T", bind.Binding.Body)
	}

	// Condition of first branch should be LtNode: x < 0
	_, ok = ifNode.Cond.(ast.LtNode)
	if !ok {
		t.Errorf("Expected condition of first branch to be LtNode, got %T", ifNode.Cond)
	}

	// Else branch should be another IfNode for otherwise
	elseIfNode, ok := ifNode.Else.(ast.IfNode)
	if !ok {
		t.Fatalf("Expected IfNode in else branch, got %T", ifNode.Else)
	}

	// Condition of otherwise should be BoolNode{Val: true}
	boolNode, ok := elseIfNode.Cond.(ast.BoolNode)
	if !ok || !boolNode.Val {
		t.Errorf("Expected BoolNode{Val: true} for otherwise, got %v", elseIfNode.Cond)
	}
}

// A definition whose RHS leaves tokens unconsumed must be rejected, not
// silently truncated. Historically the leftover was dropped, so a stranded
// operator turned `med x = True, if 1<x<10` (parsed as `1<x` on an old build)
// into a half-equation with `<10 = False, otherwise` discarded.
func TestTrailingTokensRejected(t *testing.T) {
	cases := []string{
		"is3 3 = True\n=False", // guard-less second clause (the real ~/.script.m case)
		"f x = x )",            // stray close paren
		"f x = x, 5",           // comma that is neither `, if` nor `, otherwise`
	}
	for _, src := range cases {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected parse error for %q, got none", src)
				} else if _, ok := r.(ParseError); !ok {
					t.Errorf("expected ParseError for %q, got %T", src, r)
				}
			}()
			NewParser(lexer.Tokenize(src)).Parse()
		}()
	}
}
