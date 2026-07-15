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
	// load stdenv (go test runs in the package dir; stdenv.m is one level up)
	env, typeEnv, err := LoadScriptFile("../stdenv.m", ast.NewEnv(), typecheck.DefaultTypeEnv())
	if err != nil {
		t.Fatalf("load stdenv: %v", err)
	}
	env, _, err = LoadScriptFile(path, env, typeEnv)
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

func TestOrdChr(t *testing.T) {
	src := "main = (ord 'a', chr 98, [ord c - ord '0' | c <- \"123\"])\n"
	if got := evalMain(t, src); got != "(97,'b',[1,2,3])" {
		t.Errorf("ord/chr: got %s", got)
	}
}

func TestZipWith(t *testing.T) {
	src := "main = (zipWith (\\a. \\b. a + b) [1,2,3] [10,20,30], zip2 [1,2] \"ab\")\n"
	if got := evalMain(t, src); got != "([11,22,33],[(1,'a'),(2,'b')])" {
		t.Errorf("zipWith/zip2: got %s", got)
	}
}

func TestStepRanges(t *testing.T) {
	src := "main = ([1,3..9], [10,8..0], take 4 [0,5..])\n"
	if got := evalMain(t, src); got != "([1,3,5,7,9],[10,8,6,4,2,0],[0,5,10,15])" {
		t.Errorf("step ranges: got %s", got)
	}
}

func TestLambdaPatterns(t *testing.T) {
	src := "main = (map (\\(a, b). a + b) [(1,2),(3,4)], (\\(x:xs). x) [7,8,9])\n"
	if got := evalMain(t, src); got != "([3,7],7)" {
		t.Errorf("lambda patterns: got %s", got)
	}
}

func TestOperatorSections(t *testing.T) {
	src := "main = (map (* 2) [1,2,3], filter (> 3) [1,2,3,4,5], foldl (*) 1 [1,2,3,4], map (mod 2) [10,11])\n"
	if got := evalMain(t, src); got != "([2,4,6],[4,5],24,[0,1])" {
		t.Errorf("operator sections: got %s", got)
	}
}

func TestPowAndIndex(t *testing.T) {
	// ^ : right-assoc, tighter than * ; ! : left-assoc, tighter than + but
	// looser than application
	src := "main = (2 ^ 10, 2 ^ 3 ^ 2, 2 * 3 ^ 2, [10,20,30] ! 1, \"hi\" ! 0, [10,20,30] ! 1 + 5, [[1,2],[3,4]] ! 1 ! 0, id [7,8,9] ! 2)\n"
	if got := evalMain(t, src); got != "(1024,512,18,20,'h',25,3,9)" {
		t.Errorf("^ and !: got %s", got)
	}
}

// Continued relations `a < b < c` desugar to `(a < b) & (b < c)`, chaining
// any mix of comparison operators and short-circuiting left to right.
func TestContinuedRelations(t *testing.T) {
	src := "main = [ (0 <= x < 10, 1 < x <= 5, 3 < x < x + 1, 1 <= x <= 3 <= 9) | x <- [0, 5, 10] ]\n"
	want := "[(True,False,False,False),(True,True,True,False),(False,False,True,False)]"
	if got := evalMain(t, src); got != want {
		t.Errorf("continued relations: got %s", got)
	}
	// short-circuit: the second relation is never evaluated when the first
	// fails, so a would-be error to its right is not reached
	if got := evalMain(t, "main = 5 < 3 < (1 / 0)\n"); got != "False" {
		t.Errorf("continued relation short-circuit: got %s", got)
	}
}

// Miranda-style type signatures are accepted and discarded: plain,
// multi-name, polymorphic, parenthesised-operator, and local (where) forms.
func TestTypeDeclarations(t *testing.T) {
	src := `
double :: num -> num
double x = x * 2
inc, dec :: num -> num
inc x = x + 1
dec x = x - 1
first :: (*, **) -> *
first (a, b) = a
(plus) :: num -> num -> num
plus a b = a + b
label :: [char] -> [char]
label s = tag ++ s
          where
          tag :: [char]
          tag = "> "
main = (double 21, inc 9, dec 9, first (7, 'x'), plus 3 4, label "hi")
`
	if got := evalMain(t, src); got != `(42,10,8,7,7,"> hi")` {
		t.Errorf("type declarations: got %s", got)
	}
}

func TestListPatterns(t *testing.T) {
	src := `
describe []        = "empty"
describe [x]       = "one"
describe [x, y]    = "two"
describe (x:y:xs)  = "3+"
classify [0]       = "zero"
classify [a, b]    = "pair"
classify other     = "other"
main = ([describe l | l <- [[], [1], [1,2], [1,2,3]]],
        classify [0], classify [5], classify [1,2])
`
	if got := evalMain(t, src); got != `(["empty","one","two","3+"],"zero","other","pair")` {
		t.Errorf("list patterns: got %s", got)
	}
}

func TestNonLinearPatterns(t *testing.T) {
	src := `
oeq x x = True
oeq x y = False
samepair (x, x) = 1
samepair (x, y) = 0
main = (oeq 1 1, oeq 1 2, oeq "hi" "hi", oeq [1,2] [1,3], samepair (3,3), samepair (3,4))
`
	if got := evalMain(t, src); got != "(True,False,True,False,1,0)" {
		t.Errorf("non-linear patterns: got %s", got)
	}
}
