package repl

import (
	"os"
	"path/filepath"
	"testing"

	"pkreyenhop.com/miracula-go/ast"
	"pkreyenhop.com/miracula-go/eval"
	"pkreyenhop.com/miracula-go/typecheck"
)

// evalMain loads a script and returns the printed value of its `main` binding.
func evalMain(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "prog.m")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatalf("write script: %v", err)
	}
	env, _, err := LoadScriptFile(path, ast.NewEnv(), typecheck.DefaultTypeEnv())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	val, ok := env.Lookup("main")
	if !ok {
		t.Fatalf("main not defined")
	}
	return eval.PrintNode(env, eval.Whnf(env, val))
}

// A user parameter named like the desugarer's synthetic params (p0, p1, ...)
// must not be captured by desugarer-introduced references. Before the fix this
// looped forever / returned the wrong value.
func TestSyntheticNameCollision(t *testing.T) {
	src := `
walk = go 0 0 0
       where
       go r c p1 = p1, if r >= 3
                 = go (r + 1) 0 p1, if c >= 3
                 = go r (c + 1) (p1 + r), otherwise
main = walk
`
	if got := evalMain(t, src); got != "9" {
		t.Errorf("accumulator with p1 param: expected 9, got %s", got)
	}
}

// A local (where/lambda) binding named like a native builtin must shadow it.
func TestLocalShadowsBuiltin(t *testing.T) {
	src := `
fstp (a, b) = a
outer x = fstp split + member
          where
          split = (x + 100, 0)
          member = x * 2
main = outer 5
`
	if got := evalMain(t, src); got != "115" {
		t.Errorf("local shadow of split/member: expected 115, got %s", got)
	}
}

// <, >, <=, >= are polymorphic and structural (chars, strings, tuples, bools).
func TestPolymorphicOrdering(t *testing.T) {
	cases := []struct {
		expr, want string
	}{
		{`'a' < 'b'`, "True"},
		{`"abc" < "abd"`, "True"},
		{`"ab" < "abc"`, "True"},
		{`[1,2,3] < [1,3]`, "True"},
		{`(1, 'a') < (1, 'b')`, "True"},
		{`(2, 'a') < (1, 'z')`, "False"},
		{`False < True`, "True"},
		{`'z' <= 'a'`, "False"},
	}
	for _, c := range cases {
		got := evalMain(t, "main = "+c.expr+"\n")
		if got != c.want {
			t.Errorf("%s: expected %s, got %s", c.expr, c.want, got)
		}
	}

	// natural string sort via plain <
	src := `
scmp a b = 0 - 1, if a < b
         = 1, if b < a
         = 0, otherwise
main = sort_by scmp ["pear", "apple", "fig", "banana"]
`
	if got := evalMain(t, src); got != `["apple","banana","fig","pear"]` {
		t.Errorf("string sort: got %s", got)
	}
}

func TestBitwise(t *testing.T) {
	src := "main = (xor 12 10, band 12 10, bor 12 10, shl 1 4, shr 255 4)\n"
	if got := evalMain(t, src); got != "(6,8,14,16,15)" {
		t.Errorf("bitwise: got %s", got)
	}
}

func TestMemofix(t *testing.T) {
	src := `
fib = memofix f
      where f rec n = n, if n < 2
                    = rec (n - 1) + rec (n - 2), otherwise
main = fib 40
`
	if got := evalMain(t, src); got != "102334155" {
		t.Errorf("memofix fib: got %s", got)
	}
}

func TestLetAndDestructure(t *testing.T) {
	if got := evalMain(t, "main = let x = 6; y = x * 2 in x + y\n"); got != "18" {
		t.Errorf("let: got %s", got)
	}
	src := `
divmod a b = (a / b, a mod b)
main = r where (q, r) = divmod 17 5
`
	if got := evalMain(t, src); got != "2" {
		t.Errorf("where destructure: got %s", got)
	}
}

func TestPriorityQueue(t *testing.T) {
	src := `
drain pq = if pq_null pq then []
           else let (p, v, rest) = pq_pop pq in (p, v) : drain rest
main = drain (pq_push (pq_push (pq_push pq_empty 3 30) 1 10) 2 20)
`
	if got := evalMain(t, src); got != "[(1,10),(2,20),(3,30)]" {
		t.Errorf("priority queue: got %s", got)
	}
}
