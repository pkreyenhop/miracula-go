package repl

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"pkreyenhop.com/miracula-go/ast"
	"pkreyenhop.com/miracula-go/eval"
)

func TestPerformanceFib29(t *testing.T) {
	// 1. Create a temporary script file containing the fibonacci definition
	tmpDir, err := os.MkdirTemp("", "miracula-perf-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "fib.m")
	content := []byte(`
fib 0 = 0
fib 1 = 1
fib n = fib (n-1) + fib (n-2)
`)
	if err := os.WriteFile(scriptPath, content, 0644); err != nil {
		t.Fatalf("Failed to write fib script file: %v", err)
	}

	// 2. Load the script file into the environment
	env := &ast.Env{}
	// Load standard environment if it exists in the parent directory
	stdenvPath := filepath.Join("..", "stdenv.m")
	if _, statErr := os.Stat(stdenvPath); statErr == nil {
		env, err = LoadScriptFile(stdenvPath, env)
		if err != nil {
			t.Fatalf("Failed to load stdenv.m: %v", err)
		}
	}

	env, err = LoadScriptFile(scriptPath, env)
	if err != nil {
		t.Fatalf("Failed to load fib.m: %v", err)
	}

	// 3. Define the expression we want to evaluate: "fib 29"
	expr := ast.AppNode{
		Left:  ast.VarNode{Name: "fib"},
		Right: ast.IntNode{Val: 29},
	}

	// 4. Measure evaluation time
	startTime := time.Now()
	res := eval.Whnf(env, expr)
	duration := time.Since(startTime)

	// Verify correctness of the output: fib 29 = 514229
	intRes, ok := res.(ast.IntNode)
	if !ok {
		t.Fatalf("Expected fib 29 to return IntNode, got %T (%v)", res, res)
	}
	if intRes.Val != 514229 {
		t.Fatalf("Expected fib 29 to equal 514229, got %d", intRes.Val)
	}

	// 5. Check performance against baseline + safety margin
	// Baseline: ~250 ms (current performance on dev machine)
	// Safety margin: 100% (allows up to 500 ms)
	t.Logf("fib 29 evaluation took: %v", duration)

	baseline := 250 * time.Millisecond
	safetyMarginPercent := 100
	maxAllowed := baseline * time.Duration(1+(safetyMarginPercent/100)) // 500ms

	if duration > maxAllowed {
		t.Errorf("Performance regression: fib 29 took %v, which exceeds the max allowed limit of %v (baseline: %v, safety margin: %d%%)",
			duration, maxAllowed, baseline, safetyMarginPercent)
	}
}

func BenchmarkFib25(b *testing.B) {
	stdenvPath := filepath.Join("..", "stdenv.m")
	tmpDir, err := os.MkdirTemp("", "miracula-bench-test")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "fib.m")
	content := []byte(`
fib 0 = 0
fib 1 = 1
fib n = fib (n-1) + fib (n-2)
`)
	if err := os.WriteFile(scriptPath, content, 0644); err != nil {
		b.Fatalf("Failed to write fib script file: %v", err)
	}

	env := &ast.Env{}
	if _, statErr := os.Stat(stdenvPath); statErr == nil {
		env, err = LoadScriptFile(stdenvPath, env)
		if err != nil {
			b.Fatalf("Failed to load stdenv.m: %v", err)
		}
	}

	env, err = LoadScriptFile(scriptPath, env)
	if err != nil {
		b.Fatalf("Failed to load fib.m: %v", err)
	}

	expr := ast.AppNode{
		Left:  ast.VarNode{Name: "fib"},
		Right: ast.IntNode{Val: 25},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eval.Whnf(env, expr)
	}
}
