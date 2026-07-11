package ast

import (
	"testing"
)

func TestEnvBasic(t *testing.T) {
	parent := &Env{}
	
	// Extend parent
	env1 := parent.Extend("x", IntNode{Val: 42})
	val, ok := env1.Lookup("x")
	if !ok {
		t.Fatalf("Expected variable x to be bound")
	}
	intVal, ok := val.(IntNode)
	if !ok || intVal.Val != 42 {
		t.Errorf("Expected x to be bound to IntNode(42), got %v", val)
	}

	// Lookup non-existent
	_, ok = env1.Lookup("y")
	if ok {
		t.Errorf("Did not expect variable y to be bound")
	}
}

func TestEnvScopingAndGetNames(t *testing.T) {
	parent := &Env{}
	env1 := parent.Extend("x", IntNode{Val: 1})
	env2 := env1.Extend("y", IntNode{Val: 2})
	env3 := env2.Extend("x", IntNode{Val: 3}) // shadow x

	// Lookup shadowed variable
	val, ok := env3.Lookup("x")
	if !ok {
		t.Fatalf("Expected x to be bound")
	}
	intVal, ok := val.(IntNode)
	if !ok || intVal.Val != 3 {
		t.Errorf("Expected x to lookup the shadowed value 3, got %v", val)
	}

	// Verify all unique names in scope
	names := env3.GetNames()
	expectedNames := map[string]bool{"x": true, "y": true}
	if len(names) != 2 {
		t.Errorf("Expected 2 names in environment list, got %v", names)
	}
	for _, n := range names {
		if !expectedNames[n] {
			t.Errorf("Unexpected name %s in environment list", n)
		}
	}
}
