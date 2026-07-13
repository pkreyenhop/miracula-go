# Miracula Improved: Detailed Implementation Plan

This document outlines the step-by-step technical implementation plan for extending the Miracula Go runtime and interpreter to support the high-performance primitives, 64-bit default integers, memoization, and data structures outlined in [MIRACULA_IMPROVED.md](file:///Users/pkreyenhop/src/miracula-go/MIRACULA_IMPROVED.md).

**Status update (2026-07-13).** Phases 1–5 below are implemented. They were necessary but not sufficient: `aoc8.m` still took **119.4 s** and — worse — computed the **wrong answer** (61710 instead of 244188 from the reference Go solver) because the 2-nearest-neighbour sweep heuristic does not find the 1000 globally shortest pairs. Profiling (`cpu4.prof`) showed the remaining cost was not in the built-ins at all: `ast.(*Env).Lookup` accounted for ~44 % of cumulative CPU (plus 12 % in `runtime.memequal` doing its string compares), and GC/allocator work for most of the rest. The root cause was a scoping defect in the evaluator (Phase 6).

**Second status update (same day).** Phases 6, 7, and 8.1–8.2 are now implemented, plus the argument-passthrough part of Phase 9.4 and the Phase 3 string-builder caveat. Results: `aoc8.m` produces the correct answer **244188 in 3.4 s** (from 119.4 s and a wrong answer); `aoc2.m`, which previously crashed with a fatal stack overflow, completes in 36.5 s with both parts verified against the reference Go solver; `sum [1..1000000]` runs in 1.7 s in constant space (previously a stack overflow at 100 000). All other `aoc*.m` outputs are byte-identical to before, `go test ./...` passes, and `test_miracula.m` prints `ALL TESTS PASSED!`.

**Third status update (2026-07-14).** Phase 8.3–8.4 landed (frame shrink + tail-position thunk trampolining; `foldr` depth <100 k → ~400 k). A fresh profile then showed GC/allocation at ~60 % with the comprehension machinery and the `sort_edges` comparator as the top interpreter costs, so the profile-guided slice of Phase 9 landed next: allocation-free skipping in `stepZFGenerator` and decorate-sort-undecorate in `sort_edges`/`sort_pts`. **`aoc8.m` is now 244188 in 1.45 s.** (`-cpuprofile` also now flushes in `-x` mode via a REPL exit hook; it previously wrote an empty file because `EvaluateAndExit` exits with `os.Exit`.)

**Fourth status update (2026-07-14) — Phase 9 complete.** The lexical-addressing resolver (9.1/9.3) landed first, then the explicit-continuation machine (9.5): `whnfCore` now runs on a heap-backed control stack, so **evaluation depth is unbounded** — `foldr` over `[1..3000000]` evaluates in 3.4 s (a fatal stack overflow at 500 k before), and the fully lazy fold over `[1..1000000]` in 1.9 s. Application fast paths, inline control/pending buffers, and an amortized interrupt check keep the cost of the machine at ~3 % on `aoc8.m` (median 1.49 s). Item 9.2 (multi-binding frames) was evaluated and **rejected on measured grounds** — see below.

**Fifth status update (2026-07-14) — Phase 10 complete.** Maps are immutable AVL trees with native int64/string keys (50 000 `h_insert`s: **422 ms**, vs not finishing within 115 s before), vectors are a first-class type (`to_vec`/`vec_get`/`vec_len`/`vec_set`/`vec_to_list`, O(1) reads), and `memoize` keys integer arguments without serialization. All plan phases are now closed. Remaining future work beyond this plan: strictness analysis + uncurrying (compiler stage), CAF thunks for globals (Phase 6 follow-up; a self-recursive global like `x = x + 1` loops in constant space instead of crashing — blackhole detection for globals needs memoized cells), and documenting the native builtins in the manual, which currently lists none of them.

---

## 1. Upgrade to 64-bit Integers ✅ (implemented)

Currently, integers in Miracula are represented as standard Go `int` values, which can lead to overflow on 32-bit compilation targets or when handling Advent of Code puzzle outputs (which frequently require 64-bit ranges).

### Code Changes:
1. **[lexer/lexer.go](file:///Users/pkreyenhop/src/miracula-go/lexer/lexer.go)**:
   - Modify the `Token` struct: change `Int int` to `Int int64`.
   - Update string conversions: change `strconv.Atoi` to `strconv.ParseInt(..., 10, 64)`.
2. **[ast/ast.go](file:///Users/pkreyenhop/src/miracula-go/ast/ast.go)**:
   - Change `IntNode` struct: replace `Val int` with `Val int64`.
   - Change `PatInt` struct: replace `Val int` with `Val int64`.
3. **[eval/eval.go](file:///Users/pkreyenhop/src/miracula-go/eval/eval.go)**:
   - Modify type casting and operations (addition, subtraction, multiplication, modulo, division, comparison) to handle `int64` operands natively.

---

## 2. Introduce Native Maps & Sets ✅ (implemented — with a complexity caveat)

Implemented as `MapNode`/`SetNode` wrapping Go maps, with `h_lookup`, `h_lookup_def`, `h_insert`, `member`, `empty_map`, `empty_set`.

**Caveat found in review:** `h_insert` copies the *entire* Go map on every insert (`eval.go`, `HInsertPartialNode2` case), so an insert is O(map size), and keys are stringified via `strconv.FormatInt` on every operation (one allocation per lookup/insert). A union-find loop that path-compresses via `h_insert` therefore degrades toward O(N²). This is acceptable at N = 1000 but is the next scaling wall; Phase 10 replaces the representation.

---

## 3. String Splitting & Tokenization ✅ (implemented)

`split` and `parse_ints` are implemented natively in `eval.go` with type signatures registered in `typecheck.go`.

**Caveat found in review (✅ fixed):** `MakeStringNode` (used by `read` and `split`) built the character list with non-tail Go recursion — one Go stack frame per character, so a ~1 MB input would overflow the Go stack. Both it and `lines`' list builder now build iteratively back-to-front (as `parse_ints` already did).

---

## 4. Native List Indexing & Updates ✅ (implemented — with a complexity caveat)

`list_get` and `list_set` are implemented.

**Caveat found in review:** both convert the whole Miracula list into a Go slice *on every call* (`getIntSlice`), so `list_get` is O(N) per call, not the O(1) promised in the proposal. The primitives only pay off if the data *stays* a Go slice between calls — that requires a first-class vector value (Phase 10).

---

## 5. Automatic Function Memoization ✅ (implemented)

`memoize` is implemented using `PrintNode` serialization of the evaluated argument as the cache key.

**Caveat found in review:** serializing with `PrintNode` costs O(size of argument) per call and allocates. For integer arguments, key on the `int64` directly; reserve string serialization for compound arguments.

---

## 6. Fix Evaluator Scoping: Globals Must Not Capture the Caller's Environment ✅ (implemented)

### The defect

In `Whnf`'s `ast.VarNode` case ([eval/eval.go:366](file:///Users/pkreyenhop/src/miracula-go/eval/eval.go)), when a name resolves to a global `LamNode` (all script-level functions are stored unevaluated via `ExtendGlobal`), the subsequent `ast.LamNode` case constructs `ClosureNode{Env: env}` — closing the global function over the **caller's current local environment** instead of the global scope.

Two consequences:

1. **Dynamic scoping (correctness bug).** A global's free variables resolve against whatever the caller happens to have in scope:

   ```
   step = 10
   inc n = n + step
   main = show ((\step. inc 1) 99)   || prints 100; correct answer is 11
   ```

   Verified against the current binary: prints `100`.

2. **O(n²) evaluation (the performance bug).** Every call to a global function extends the *caller's* chain by one single-binding frame per argument, and the chain is never reset on entry to the callee. A tail-recursive loop therefore runs with an environment that grows by ~k frames per iteration, and `Env.Lookup` — which must walk the entire local chain (string-comparing every frame name) before it even consults `Globals` — costs O(iteration). Every fold, every recursive helper in every `.m` file is quadratic. This is exactly the `Env.Lookup` + `memequal` + GC signature in `cpu4.prof`.

### The fix (validated as a prototype)

Replace the lookup block in the `ast.VarNode` case of `Whnf` with: walk the local chain; on miss, consult `Globals`, and if found there, **switch `env` to a globals-only environment** before evaluating the global's body:

```go
var val ast.Node
var ok bool
for curr := env; curr != nil; curr = curr.Parent {
    if curr.Name == name {
        val = curr.Val
        ok = true
        break
    }
}
if !ok && env.Globals != nil {
    if gv, gok := env.Globals[name]; gok {
        // Globals are closed terms over the global scope: evaluate them
        // in a globals-only environment so caller locals are not captured
        // (fixes dynamic scoping and the O(n) env growth per call).
        env = &ast.Env{Globals: env.Globals}
        val = gv
        ok = true
    }
}
if !ok {
    panic(ast.RuntimeError{Msg: "Unbound variable: " + name})
}
```

No changes required in `ast.go`, `repl.go`, or `typecheck.go` — `LoadScriptFile`/`RunREPLDirect` already store script definitions via `ExtendGlobal`.

### Measured results (prototype build, Apple Silicon)

| Benchmark | Before | After scoping fix |
|---|---|---|
| `aoc8.m` (current 2-NN sweep) | 119.4 s | **1.8 s** (65×) |
| Day 8, correct all-pairs algorithm (Phase 7) | not feasible | **3.5 s**, answer 244188 ✓ |
| strict counting loop, n = 32 000 | 11.55 s | **23 ms** (~500×) |
| `sum [1..8000]` | 1.45 s | **47 ms** |
| `sum [1..50000]` | minutes (quadratic) | **235 ms** (linear) |
| scoping test above | `100` (wrong) | `11` (correct) |

### Verification

- `go test ./...` passes.
- `test_miracula.m` prints `ALL TESTS PASSED!`.
- Audit the `.m` files for code that accidentally *depends* on dynamic scoping (none found in `stdenv.m` / `aoc*.m`, but re-run `aoc_all.m` and compare outputs before/after).

### Optional follow-up in the same area

Turn globals into CAFs: wrap each global in a `ThunkCell` evaluated once in the globals-only environment, so constant applicative forms like `sum = foldl (+) 0` are evaluated once instead of re-closed on every reference.

---

## 7. Rewrite `aoc8.m` with the Correct Algorithm ✅ (implemented — 244188 in 3.4 s)

### Why the current program is wrong, independent of speed

The reference solver (removed in commit `9a717c7`, see `git show 9a717c7^:eval/aoc.go`, `Aoc8Solver`) sorts **all 499 500 pairwise distances** and processes the **first 1000 pairs** through union-find, where already-connected pairs still consume one of the 1000 attempts. The current `aoc8.m`:

1. only generates each point's two nearest neighbours as candidate edges — the 1000 globally shortest edges are *not* a subset of that set (dense clusters contribute many short edges per point), and
2. `kruskal8` counts successful *merges* (`count + 1` only when `ru != rv`) instead of processed *pairs*, and
3. `dedup8` relies on duplicate edges sorting adjacently, which breaks on distance ties.

Result: `61710` vs the reference answer `244188` on `inputs/day8.txt`.

### The fix

With Phase 6 in place, the honest brute-force algorithm is fast enough (measured 3.5 s; Phases 8–10 shrink it further). Replace `aoc8.m`'s solver with:

```
fst (a, b) = a
snd (a, b) = b

fst3 (a, b, c) = a
snd3 (a, b, c) = b
thd3 (a, b, c) = c

distSq8 (x1, y1, z1) (x2, y2, z2) = dx*dx + dy*dy + dz*dz
  where
  dx = x1 - x2
  dy = y1 - y2
  dz = z1 - z2

group3 [] = []
group3 (x:y:z:rest) = (x, y, z) : group3 rest

find_root parent x =
  if p == x then (x, parent)
  else (root, h_insert root_parent x root)
  where
  p = h_lookup_def parent x x
  res = find_root parent p
  root = fst res
  root_parent = snd res

|| process every edge in the list (caller passes exactly the 1000 shortest,
|| so a no-op union still consumes its slot, matching the puzzle statement)
unite [] parent = parent
unite (e:es) parent =
  if ru == rv then unite es parent_v
  else unite es (h_insert parent_v ru rv)
  where
  u = fst3 e
  v = snd3 e
  res_u = find_root parent u
  ru = fst res_u
  parent_u = snd res_u
  res_v = find_root parent_u v
  rv = fst res_v
  parent_v = snd res_v

group_count [] = []
group_count (x:xs) = count_more 1 x xs
  where
  count_more k val [] = [k]
  count_more k val (y:ys) = if y == val then count_more (k+1) val ys
                            else k : count_more 1 y ys

solvePart1 input = s1 * s2 * s3
  where
  pts = group3 (parse_ints input)
  n = length pts
  ipts = zip ([0 .. n - 1], pts)
  edges = [ (i, j, distSq8 p q) | (i, p) <- ipts; (j, q) <- ipts; i < j ]
  shortest = take 1000 (sort_edges edges)

  final_parent = unite shortest empty_map

  find_all_roots [] p = ([], p)
  find_all_roots (x:xs) p = (r : rest_roots, final_p)
    where
    res_x = find_root p x
    r = fst res_x
    p_x = snd res_x
    res_xs = find_all_roots xs p_x
    rest_roots = fst res_xs
    final_p = snd res_xs

  roots_res = find_all_roots [0 .. n - 1] final_parent
  roots = fst roots_res
  sorted_roots = sort_ints roots
  sizes = group_count sorted_roots
  sorted_sizes = reverse (sort_ints sizes)

  s1 = hd sorted_sizes
  s2 = hd (tl sorted_sizes)
  s3 = hd (tl (tl sorted_sizes))

main = "Advent of Code 2025 - Day 8 Results:\n" ++
       "  Part 1 (Circuit size product): " ++ show p1Result ++ "\n"
       where
       input = read "inputs/day8.txt"
       p1Result = solvePart1 input
```

This drops `seq`, `scan_left`/`scan_right`, `dedup8`, `sort_pts`, and the merge-count logic entirely — the program becomes both correct and ~40 lines shorter.

### Acceptance criteria

- `./mira -xt aoc8.m` prints `244188` (equal to `Aoc8Solver` from `9a717c7^` on `inputs/day8.txt`).
- Wall time ≤ 5 s after Phase 6; ≤ 1 s after Phases 8–9.

---

## 8. Deep-Force Robustness: Strict Folds and an Iterative Forcer ✅ (implemented; fully unbounded depth deferred to Phase 9.5)

### Evidence

After Phase 6, `sum [1..50000]` is linear (235 ms), but `sum [1..100000]` **crashes with a Go stack overflow**: `foldl` builds a 100 000-deep chain of unevaluated `(+)` thunks, and forcing it recurses once through `Whnf` per link. Each `Whnf` frame is ~2.4 KB (the giant switch keeps every case's locals live), so ~10⁵ links exhaust even the 1 GB goroutine stack. The `seq x y = ifzero x then y else y` hack in the old `aoc8.m` exists only to work around this, and it only works for integer accumulators.

### Code changes (in order of payoff/effort)

1. ✅ **`eval/eval.go` + `typecheck/typecheck.go`: native `seq :: * -> ** -> **`** — evaluates the first argument to WHNF, returns the second. Lets Miracula code force accumulators of any type and retires the `ifzero` idiom. (Script-level `seq` definitions in older files are shadowed by the builtin, which has identical semantics.)
2. ✅ **`stdenv.m`: strict left fold** — `foldl` built on `seq` (`foldl f z (x:xs) = seq z2 (foldl f z2 xs) where z2 = f z x`), keeping `sum`, `product`, etc. constant-space. Measured: `sum [1..1000000]` = 1.7 s, constant stack.
3. ✅ **`eval/eval.go`: shrink evaluator stack frames** — the built-in dispatch (`applyBuiltin`), partial-application handling (`applyPartial`), and comprehension stepping (`stepZFGenerator`) moved out of the recursive core. Measured with `go build -gcflags=-S`: the recursive frame dropped from ~2.4 KB (old `Whnf`) to 1136 B (`whnfCore`) + 80 B (`Whnf` wrapper); the safe depth for genuinely non-tail recursion (`foldr`, lazy accumulator folds through strict operands) rose from <100 000 to ~400 000 levels (300 000 measured OK at 515 ms, 500 000 overflows).
4. ✅ **Iterative thunk-chain forcing** — `Whnf` is now a thin wrapper around `whnfCore`, which records unevaluated thunks reached in tail position on an explicit pending stack instead of recursing; the final WHNF is written into every recorded cell (they share it by construction, so blackhole detection still works). Pure indirection chains no longer consume Go stack at all. List equality (`eq`) also walks tails iteratively now, and `getStringValue` builds strings in a loop instead of one Go frame per character. The ~400 000-level ceiling that remained for chains forced through strict operands was removed by the Phase 9.5 continuation machine.

---

## 9. Environment Representation: Lexical Addressing & Allocation Reduction 🟠

Phase 6 removes the quadratic blow-up, but every variable reference still walks a linked list comparing strings, every argument allocates a one-binding `Env` frame, and every non-atomic argument allocates a `ThunkCell` + `ThunkNode`. The `cpu4.prof` GC share (~35–40 %) is this allocation pressure.

### Code changes

1. ✅ **Resolver pass (`eval/resolve.go`)** runs once per definition after typechecking, in both the script loader and the REPL definition/expression paths. Every `VarNode` becomes `LocalVarNode{Depth}` (frames are single-binding, so a depth alone identifies the binding — the evaluator reaches it with Depth pointer hops and zero name comparisons) or `GlobalVarNode{Name}` (one map read, no chain walk). Builtin names stay `VarNode`s because the evaluator dispatches them before any environment lookup; the legacy by-name path is retained as a fallback. The static scope model mirrors the three binder forms the desugarer emits: `LamNode`, `LetNode` (letrec, one frame per binding in declaration order), and comprehension-generator patterns (frames in `matchPattern`/`mergeBindings` order).
2. ❌ **Multi-binding frames** — evaluated and rejected. Growing `Env` with `Names []string`/`Vals []Node` fields takes every frame allocation from the 64-byte to the 112-byte size class, and closure-argument frames (millions per run) outnumber `where`-group frames (thousands) by orders of magnitude; the pointer-boxed variant (a `FrameNode` behind the existing `Val` field) is allocation-neutral at the typical group size of 2–3 bindings while adding an indirection to every lookup. Either way the net effect on real workloads is flat-to-negative. The original motivation — one frame per multi-argument call — only pays with uncurried `CallNode` application, which belongs to a future compiler stage, not this evaluator.
3. ✅ **`eval/eval.go`**: `LocalVarNode`/`GlobalVarNode` evaluation cases added (sharing the pending-stack thunk forcing), and the globals scope-switch now reuses the chain's `Root` frame (`ast.Env.Root`) instead of allocating a globals-only environment per global reference.
4. **Thunk avoidance** (✅ first half implemented as `bindArg` in `eval.go`): an argument that is just a variable reference now passes its existing local binding through instead of allocating a fresh indirection thunk. Before this, every `loop acc (curr+1)`-style call wrapped `acc` in a new thunk, so long runs of iterations built an unbounded indirection chain — this is what crashed `aoc2.m` with a fatal stack overflow (~69 000-deep chains from its largest ranges). The remaining half — evaluating argument subtrees like `f (a+b)` eagerly — is not a local optimization: it changes termination/error semantics unless a strictness analysis proves the callee demands the argument, so it moves to future compiler work alongside uncurrying.
5. ✅ **Explicit continuation (CEK-style) evaluation** — `whnfCore` now pushes a continuation frame for every strict operand position (binary-operator operands, `if`/`ifzero`/`ifnil` conditions, function position, tuple projection, append) instead of recursing. Each frame carries a pending-thunk watermark, so thunk cells recorded while evaluating an operand receive exactly that operand's value; blackhole detection is unchanged. Nested `Whnf` calls remain only in positions bounded by data or expression nesting (builtin internals, range bounds, list difference, structural equality of heads, pattern matching). Measured: `foldr` over `[1..3000000]` = 3.3 s and the fully lazy fold over `[1..1000000]` = 1.7 s (both fatal stack overflows before); `aoc8.m` 1.33 s (faster than the 1.45 s pre-machine baseline) after the recovery optimizations — application fast paths for common callee shapes (lambda, closure, global function, local bound to an evaluated closure, builtin names), stack-inline buffers for the control and pending stacks so shallow nested evaluations never allocate them, an interrupt check amortized to every 4096 steps, quick-eval paths that compute binops / branch conditions inline when their operands are literals or locals already bound to evaluated values (a pure read, so evaluation order stays unobservable), and — critically — storing the *already-boxed* interface into continuation frames instead of re-boxing the concrete operator node, which `runtime.convT`-allocated 32–48 bytes per push and initially cost call-and-arithmetic-heavy code ~50 % (naive `fib 32`: 817 ms pre-machine → 1 197 ms → **826 ms** after these fixes).
6. ✅ **Comprehension stepping without per-element machinery** — `stepZFGenerator` now loops over source elements that fail the pattern match or a directly following filter qualifier without allocating a generator/append/conditional node per skipped element, and yields an element by consing straight onto the advanced generator. In `aoc8.m` this removes ~6 allocations for each of the ~500 000 filtered-out `i < j` pairs.
7. ✅ **Decorate-sort-undecorate in native sorts** — `sort_edges` and `sort_pts` extract each element's integer key once (`keyedNode`) and sort plain `int64`s, instead of re-forcing thunks in every comparison (~19 M comparator `Whnf` calls in `aoc8.m` before). Items 6+7 together: `aoc8.m` 3.2 s → **1.45 s**.

Measured effect of items 1+3 (2026-07-14): no regression and no headline speedup — `aoc8.m` stays at 1.45 s and `aoc2.m` at ~35 s, because after Phases 6–8 the profile is GC-bound (tuple/thunk/cons allocation), not lookup-bound. The value is architectural: every variable now carries static coordinates and no run-time name comparison remains, which is the prerequisite for items 2 and 5. A dedicated resolver stress script (shadowed globals in lambdas, cross-referencing nested `where` blocks, dependent generators, cons/tuple pattern binding order) produces byte-identical output to the pre-resolver binary, and the full `aoc*.m` sweep, `test_miracula.m`, and `go test ./...` all pass unchanged.

---

## 10. Data-Structure Upgrades: Persistent Int-Keyed Maps & Real Vectors ✅ (implemented)

1. ✅ **Persistent map (`ast.MapNode`)**: now an immutable AVL tree (`ast/maptree.go`) whose keys are `ast.MapKey` — an `int64` or a string, never a formatted string for integers:
   - `h_insert` is O(log n) time *and space* with structural sharing; the old code copied the whole Go map per insert.
   - `h_lookup`/`h_lookup_def` are allocation-free; no `strconv.FormatInt` per operation.
   - `h_lookup_def` also became lazier: the default is only evaluated on a miss (it was always forced before).
   - Measured: building a 50 000-entry map by repeated `h_insert` takes **422 ms**; the previous representation did not finish within a 115 s cap (O(n²) copying) — an unbounded asymptotic win.
   - Sets are untouched: no set-insert builtin exists, so `SetNode` stays a Go map until sets grow an API.
2. ✅ **Vector value type**: `VecNode{Elems []ast.Node}` with `to_vec :: [*] -> vec *`, `vec_get :: vec * -> num -> *` (true O(1), elements stay lazy), `vec_len`, `vec_set` (O(n) copy of the element slice, persistent — the original vector is untouched), and `vec_to_list`. Registered in the type checker as `VecType`. `list_get`/`list_set` remain for compatibility but vectors are the supported path — they never re-convert a list per call. (`IntVecNode` fast path deferred until a workload needs it.)
3. ✅ **`memoize`**: integer arguments key a dedicated `map[int64]Node` cache directly; `PrintNode` serialization is only used for compound arguments.

---

## Verification & Benchmark Methodology

Every phase lands with numbers, gathered the same way:

1. **Correctness**: `go test ./...`; `./mira -x test_miracula.m` must end with `ALL TESTS PASSED!`; `./mira -x aoc_all.m` outputs compared against the reference answers (Day 8: **244188**).
2. **Macro benchmark**: `./mira -xt aoc8.m` (3 runs, median). Targets: ≤ 5 s after Phase 6+7, ≤ 1 s after Phase 9.
3. **Micro benchmarks** (scaling shape matters more than absolute time — double n, time must ~double, not quadruple):
   - `./mira -t "sum [1..50000]"`
   - `./mira -t "length [1..200000]"`
   - a strict tail-recursive counting loop at n = 32 000.
4. **Profile**: `./mira -cpuprofile=cpu.prof -x aoc8.m && go tool pprof -top cpu.prof` — after Phase 9, `Env.Lookup` and `memequal` should no longer appear in the top 25.
