package typecheck

import (
	"testing"
	"pkreyenhop.com/miracula-go/ast"
)

func TestTypeCheckerSimple(t *testing.T) {
	tc := NewTypeChecker()
	env := DefaultTypeEnv()

	// Test 1: 10 + 5 : Int
	node1 := ast.AddNode{Left: ast.IntNode{Val: 10}, Right: ast.IntNode{Val: 5}}
	ty1, _, err := tc.Infer(env, node1, nil)
	if err != nil {
		t.Fatalf("Failed to infer 10 + 5: %v", err)
	}
	if ty1 != TInt {
		t.Errorf("Expected Int, got %v", ty1)
	}

	// Test 2: 10 == 10 : Bool
	node2 := ast.EqNode{Left: ast.IntNode{Val: 10}, Right: ast.IntNode{Val: 10}}
	ty2, _, err := tc.Infer(env, node2, nil)
	if err != nil {
		t.Fatalf("Failed to infer 10 == 10: %v", err)
	}
	if ty2 != TBool {
		t.Errorf("Expected Bool, got %v", ty2)
	}

	// Test 3: 10 == True : Type Error wrapped in TypeError
	node3 := ast.EqNode{Left: ast.IntNode{Val: 10}, Right: ast.BoolNode{Val: true}}
	_, _, err = tc.Infer(env, node3, nil)
	if err == nil {
		t.Errorf("Expected type error for 10 == True, but type checking passed")
	} else {
		te, ok := err.(*TypeError)
		if !ok {
			t.Errorf("Expected error to be *TypeError, got %T (%v)", err, err)
		} else if te.Node != node3 {
			t.Errorf("Expected TypeError Node to be the offending EqNode, got %T (%v)", te.Node, te.Node)
		}
	}
}
