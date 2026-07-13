package repl

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func TestEditorFilenameValidation(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"/e script.m", true},
		{"/e other.m", true},
		{"/e test.m", true},
		{"/e test.txt", false},
		{"/e test.m.txt", false},
		{"/e test", false},
	}

	for _, tt := range tests {
		parts := strings.Fields(tt.input)
		target := parts[1]
		isValid := strings.HasSuffix(target, ".m")
		if isValid != tt.expected {
			t.Errorf("Validation for %q expected %v, got %v", tt.input, tt.expected, isValid)
		}
	}
}

func TestErrorTracking(t *testing.T) {
	// Create a temporary script file
	tmpDir, err := os.MkdirTemp("", "miracula-repl-err-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test 1: Type error tracking
	scriptPath := filepath.Join(tmpDir, "type_err.m")
	content := []byte("x = 5 == \"hello\"\n")
	if err := os.WriteFile(scriptPath, content, 0644); err != nil {
		t.Fatalf("Failed to write type_err.m: %v", err)
	}

	env := ast.NewEnv()
	typeEnv := typecheck.DefaultTypeEnv()
	_, _, err = LoadScriptFile(scriptPath, env, typeEnv)
	if err == nil {
		t.Fatalf("Expected type error but LoadScriptFile succeeded")
	}

	if lastErrorLine != 1 {
		t.Errorf("Expected lastErrorLine to be 1 for type error, got %d", lastErrorLine)
	}
	if lastErrorCol != 5 {
		t.Errorf("Expected lastErrorCol to be 5 for type error (at ==), got %d", lastErrorCol)
	}

	// Test 2: Parse error tracking
	scriptPath2 := filepath.Join(tmpDir, "parse_err.m")
	content2 := []byte("x = +\n")
	if err := os.WriteFile(scriptPath2, content2, 0644); err != nil {
		t.Fatalf("Failed to write parse_err.m: %v", err)
	}

	_, _, err = LoadScriptFile(scriptPath2, env, typeEnv)
	if err == nil {
		t.Fatalf("Expected parse error but LoadScriptFile succeeded")
	}

	if lastErrorLine != 1 {
		t.Errorf("Expected lastErrorLine to be 1 for parse error, got %d", lastErrorLine)
	}
	if lastErrorCol != 5 {
		t.Errorf("Expected lastErrorCol to be 5 for parse error (at +), got %d", lastErrorCol)
	}
}

func TestExpandHome(t *testing.T) {
	origHome, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Skipping TestExpandHome: user home dir not available")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"~", origHome},
		{"~/foo", filepath.Join(origHome, "foo")},
		{"~/.script.m", filepath.Join(origHome, ".script.m")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		res := ExpandHome(tt.input)
		if res != tt.expected {
			t.Errorf("ExpandHome(%q) = %q, expected %q", tt.input, res, tt.expected)
		}
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		content  string
		expected int
	}{
		{"", 0},
		{"hello", 1},
		{"hello\n", 1},
		{"hello\nworld", 2},
		{"hello\nworld\n", 2},
	}

	for _, tt := range tests {
		res := countLines(tt.content)
		if res != tt.expected {
			t.Errorf("countLines(%q) = %d, expected %d", tt.content, res, tt.expected)
		}
	}
}

func TestHandleShowDefinition(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "miracula-lookup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "source.m")
	fileContent := "add x y = x + y\nsub x y = x - y\n"
	if err := os.WriteFile(filePath, []byte(fileContent), 0644); err != nil {
		t.Fatalf("Failed to write temp source file: %v", err)
	}

	env := ast.NewEnv()
	nodeAdd := ast.IntNode{Val: 1}
	nodeSub := ast.IntNode{Val: 2}

	env = env.Extend("add", nodeAdd)
	env = env.Extend("sub", nodeSub)

	boxedAdd, _ := env.Lookup("add")
	boxedSub, _ := env.Lookup("sub")

	ast.NodePositions.Store(ast.GetNodeKey(boxedAdd), ast.Position{Filename: filePath, Line: 1, Col: 1})
	ast.NodePositions.Store(ast.GetNodeKey(boxedSub), ast.Position{Filename: filePath, Line: 2, Col: 1})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	handleShowDefinition(env, "add", false)
	handleShowDefinition(env, "sub", false)
	handleShowDefinition(env, "missing", false)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	expectedAdd := fmt.Sprintf("%s:1: add x y = x + y", filePath)
	expectedSub := fmt.Sprintf("%s:2: sub x y = x - y", filePath)
	expectedMissing := "Function 'missing' not found."

	if !strings.Contains(output, expectedAdd) {
		t.Errorf("Expected output to contain %q, got:\n%s", expectedAdd, output)
	}
	if !strings.Contains(output, expectedSub) {
		t.Errorf("Expected output to contain %q, got:\n%s", expectedSub, output)
	}
	if !strings.Contains(output, expectedMissing) {
		t.Errorf("Expected output to contain %q, got:\n%s", expectedMissing, output)
	}
}

func TestGetManualContent(t *testing.T) {
	content := getManualContent()
	if !strings.Contains(content, "Miracula System Manual") {
		t.Errorf("Expected manual content to contain 'Miracula System Manual'")
	}
	if !strings.Contains(content, "How to use") {
		t.Errorf("Expected manual content to contain 'How to use'")
	}
}

func TestHandleShellCommand(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	handleShellCommand("!echo hello shell command")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "hello shell command") {
		t.Errorf("Expected output to contain 'hello shell command', got %q", output)
	}
}
