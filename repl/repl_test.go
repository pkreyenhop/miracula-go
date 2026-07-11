package repl

import (
	"os"
	"path/filepath"
	"testing"
	"pkreyenhop.com/miracula-go/ast"
	"pkreyenhop.com/miracula-go/eval"
	"pkreyenhop.com/miracula-go/typecheck"
)

func TestIsWordChar(t *testing.T) {
	tests := []struct {
		char     rune
		expected bool
	}{
		{'a', true},
		{'Z', true},
		{'0', true},
		{'_', true},
		{'-', false},
		{' ', false},
		{'\t', false},
	}

	for _, tt := range tests {
		res := isWordChar(tt.char)
		if res != tt.expected {
			t.Errorf("isWordChar(%q) expected %v, got %v", tt.char, tt.expected, res)
		}
	}
}

func TestLoadScriptFile(t *testing.T) {
	// Create a temporary script file
	tmpDir, err := os.MkdirTemp("", "miracula-repl-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "test.m")
	content := []byte(`
|| Standard test bindings
x = 42
y = 100
`)
	if err := os.WriteFile(scriptPath, content, 0644); err != nil {
		t.Fatalf("Failed to write test script file: %v", err)
	}

	env := ast.NewEnv()
	typeEnv := typecheck.DefaultTypeEnv()
	loadedEnv, _, err := LoadScriptFile(scriptPath, env, typeEnv)
	if err != nil {
		t.Fatalf("LoadScriptFile failed: %v", err)
	}

	// Verify loaded bindings
	valX, ok := loadedEnv.Lookup("x")
	if !ok {
		t.Fatalf("Expected x to be defined in loaded env")
	}
	xEval := eval.Whnf(loadedEnv, valX)
	intX, ok := xEval.(ast.IntNode)
	if !ok || intX.Val != 42 {
		t.Errorf("Expected x to evaluate to 42, got %v", xEval)
	}

	valY, ok := loadedEnv.Lookup("y")
	if !ok {
		t.Fatalf("Expected y to be defined in loaded env")
	}
	yEval := eval.Whnf(loadedEnv, valY)
	intY, ok := yEval.(ast.IntNode)
	if !ok || intY.Val != 100 {
		t.Errorf("Expected y to evaluate to 100, got %v", yEval)
	}
}
