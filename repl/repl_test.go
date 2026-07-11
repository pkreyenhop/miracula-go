package repl

import (
	"fmt"
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

func TestHistoryFile(t *testing.T) {
	// Backup history.m if it exists
	var backedUp bool
	if _, err := os.Stat("history.m"); err == nil {
		_ = os.Rename("history.m", "history.m.bak")
		backedUp = true
	}
	defer func() {
		_ = os.Remove("history.m")
		if backedUp {
			_ = os.Rename("history.m.bak", "history.m")
		}
	}()

	// 1. Generate 250 dummy history commands
	var testHistory []string
	for i := 0; i < 250; i++ {
		testHistory = append(testHistory, fmt.Sprintf("cmd_%d", i))
	}

	// 2. Save history
	saveHistory(testHistory)

	// 3. Load history back
	loaded := loadHistory()

	// 4. Assertions
	if len(loaded) != 200 {
		t.Errorf("Expected loaded history to be capped at 200 elements, got %d", len(loaded))
	}

	// Should contain the last 200 elements: cmd_50 to cmd_249
	if len(loaded) > 0 {
		if loaded[0] != "cmd_50" {
			t.Errorf("Expected first element of loaded history to be cmd_50, got %s", loaded[0])
		}
		if loaded[len(loaded)-1] != "cmd_249" {
			t.Errorf("Expected last element of loaded history to be cmd_249, got %s", loaded[len(loaded)-1])
		}
	}
}

func TestSaveReplDefinitions(t *testing.T) {
	// Create a temporary script file
	tmpDir, err := os.MkdirTemp("", "miracula-repl-save-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "temp_script.m")
	content := []byte("orig_val = 100\n")
	if err := os.WriteFile(scriptPath, content, 0644); err != nil {
		t.Fatalf("Failed to write temp script file: %v", err)
	}

	// 1. Load initial environment
	env := ast.NewEnv()
	typeEnv := typecheck.DefaultTypeEnv()
	loadedEnv, _, err := LoadScriptFile(scriptPath, env, typeEnv)
	if err != nil {
		t.Fatalf("Failed to load initial script file: %v", err)
	}

	// Verify original value
	valOrig, ok := loadedEnv.Lookup("orig_val")
	if !ok {
		t.Fatalf("Expected orig_val to be defined")
	}
	origEval := eval.Whnf(loadedEnv, valOrig)
	intOrig, ok := origEval.(ast.IntNode)
	if !ok || intOrig.Val != 100 {
		t.Errorf("Expected orig_val to evaluate to 100, got %v", origEval)
	}

	// 2. Simulate REPL appending a definition to the script file
	f, err := os.OpenFile(scriptPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open script file for append: %v", err)
	}
	newDef := "repl_val = 999\n"
	_, _ = f.WriteString(newDef)
	_ = f.Close()

	// 3. Reload environment from the script file (simulating REPL restart)
	reloadedEnv, _, err := LoadScriptFile(scriptPath, ast.NewEnv(), typecheck.DefaultTypeEnv())
	if err != nil {
		t.Fatalf("Failed to reload script file: %v", err)
	}

	// Verify new binding exists and has the correct value
	valNew, ok := reloadedEnv.Lookup("repl_val")
	if !ok {
		t.Fatalf("Expected repl_val to be defined after reload")
	}
	newEval := eval.Whnf(reloadedEnv, valNew)
	intNew, ok := newEval.(ast.IntNode)
	if !ok || intNew.Val != 999 {
		t.Errorf("Expected repl_val to evaluate to 999, got %v", newEval)
	}
}
