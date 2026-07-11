package eval

import (
	"testing"
	"pkreyenhop.com/miracula-go/ast"
)

func TestWhnfArithmetic(t *testing.T) {
	env := &ast.Env{}
	
	// Test: 10 + 5
	add := ast.AddNode{Left: ast.IntNode{Val: 10}, Right: ast.IntNode{Val: 5}}
	res := Whnf(env, add)
	i, ok := res.(ast.IntNode)
	if !ok || i.Val != 15 {
		t.Errorf("Expected 15, got %v", res)
	}

	// Test: 10 - 3
	sub := ast.SubNode{Left: ast.IntNode{Val: 10}, Right: ast.IntNode{Val: 3}}
	res = Whnf(env, sub)
	i, ok = res.(ast.IntNode)
	if !ok || i.Val != 7 {
		t.Errorf("Expected 7, got %v", res)
	}

	// Test: 3 * 4
	mul := ast.MulNode{Left: ast.IntNode{Val: 3}, Right: ast.IntNode{Val: 4}}
	res = Whnf(env, mul)
	i, ok = res.(ast.IntNode)
	if !ok || i.Val != 12 {
		t.Errorf("Expected 12, got %v", res)
	}

	// Test: 11 / 3 (SML division truncates towards negative infinity: 11 / 3 = 3)
	div := ast.DivNode{Left: ast.IntNode{Val: 11}, Right: ast.IntNode{Val: 3}}
	res = Whnf(env, div)
	i, ok = res.(ast.IntNode)
	if !ok || i.Val != 3 {
		t.Errorf("Expected SML div 11 / 3 = 3, got %v", res)
	}

	// Test: -11 / 3 (SML division: -11 / 3 = -4)
	divNeg := ast.DivNode{Left: ast.IntNode{Val: -11}, Right: ast.IntNode{Val: 3}}
	res = Whnf(env, divNeg)
	i, ok = res.(ast.IntNode)
	if !ok || i.Val != -4 {
		t.Errorf("Expected SML div -11 / 3 = -4, got %v", res)
	}

	// Test: 11 mod 3 (SML modulo: 11 mod 3 = 2)
	mod := ast.ModNode{Left: ast.IntNode{Val: 11}, Right: ast.IntNode{Val: 3}}
	res = Whnf(env, mod)
	i, ok = res.(ast.IntNode)
	if !ok || i.Val != 2 {
		t.Errorf("Expected SML mod 11 mod 3 = 2, got %v", res)
	}
}

func TestWhnfLazyLists(t *testing.T) {
	env := &ast.Env{}

	// Test Range: [1..5]
	rng := ast.RangeNode{Start: ast.IntNode{Val: 1}, End: ast.IntNode{Val: 5}}
	res := Whnf(env, rng)
	cons, ok := res.(ast.ConsNode)
	if !ok {
		t.Fatalf("Expected Range to evaluate to ConsNode, got %T", res)
	}

	hVal := Whnf(env, cons.Head)
	hInt, ok := hVal.(ast.IntNode)
	if !ok || hInt.Val != 1 {
		t.Errorf("Expected head to be IntNode(1), got %v", hVal)
	}

	// Evaluate tail
	tVal := Whnf(env, cons.Tail)
	tCons, ok := tVal.(ast.ConsNode)
	if !ok {
		t.Fatalf("Expected tail to evaluate to ConsNode, got %T", tVal)
	}

	thVal := Whnf(env, tCons.Head)
	thInt, ok := thVal.(ast.IntNode)
	if !ok || thInt.Val != 2 {
		t.Errorf("Expected second element to be IntNode(2), got %v", thVal)
	}
}

func TestBuiltins(t *testing.T) {
	env := &ast.Env{}

	// Test length [1..5]
	lengthApp := ast.AppNode{
		Left:  ast.VarNode{Name: "length"},
		Right: ast.RangeNode{Start: ast.IntNode{Val: 1}, End: ast.IntNode{Val: 5}},
	}
	res := Whnf(env, lengthApp)
	i, ok := res.(ast.IntNode)
	if !ok || i.Val != 5 {
		t.Errorf("Expected length [1..5] = 5, got %v", res)
	}

	// Test hd [1..5]
	hdApp := ast.AppNode{
		Left:  ast.VarNode{Name: "hd"},
		Right: ast.RangeNode{Start: ast.IntNode{Val: 1}, End: ast.IntNode{Val: 5}},
	}
	res = Whnf(env, hdApp)
	i, ok = res.(ast.IntNode)
	if !ok || i.Val != 1 {
		t.Errorf("Expected hd [1..5] = 1, got %v", res)
	}
}
