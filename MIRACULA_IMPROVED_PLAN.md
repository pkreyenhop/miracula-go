# Miracula Improved: Detailed Implementation Plan

This document outlines the step-by-step technical implementation plan for extending the Miracula Go runtime and interpreter to support the high-performance primitives, 64-bit default integers, memoization, and data structures outlined in [MIRACULA_IMPROVED.md](file:///Users/pkreyenhop/src/miracula-go/MIRACULA_IMPROVED.md).

**Status update (2026-07-13).** Phases 1–5 below are implemented. They were necessary but not sufficient: `aoc8.m` still took **119.4 s** and — worse — computed the **wrong answer** (61710 instead of 244188 from the reference Go solver) because the 2-nearest-neighbour sweep heuristic does not find the 1000 globally shortest pairs. Profiling (`cpu4.prof`) showed the remaining cost was not in the built-ins at all: `ast.(*Env).Lookup` accounted for ~44 % of cumulative CPU (plus 12 % in `runtime.memequal` doing its string compares), and GC/allocator work for most of the rest. The root cause was a scoping defect in the evaluator (Phase 6).

**Second status update (same day).** Phases 6, 7, and 8.1–8.2 are now implemented, plus the argument-passthrough part of Phase 9.4 and the Phase 3 string-builder caveat. Results: `aoc8.m` produces the correct answer **244188 in 3.4 s** (from 119.4 s and a wrong answer); `aoc2.m`, which previously crashed with a fatal stack overflow, completes in 36.5 s with both parts verified against the reference Go solver; `sum [1..1000000]` runs in 1.7 s in constant space (previously a stack overflow at 100 000). All other `aoc*.m` outputs are byte-identical to before, `go test ./...` passes, and `test_miracula.m` prints `ALL TESTS PASSED!`. Remaining open work: Phase 8.3–8.4, Phase 9.1–9.3, Phase 10.

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

## 8. Deep-Force Robustness: Strict Folds and an Iterative Forcer ✅ items 1–2 / 🟠 items 3–4

### Evidence

After Phase 6, `sum [1..50000]` is linear (235 ms), but `sum [1..100000]` **crashes with a Go stack overflow**: `foldl` builds a 100 000-deep chain of unevaluated `(+)` thunks, and forcing it recurses once through `Whnf` per link. Each `Whnf` frame is ~2.4 KB (the giant switch keeps every case's locals live), so ~10⁵ links exhaust even the 1 GB goroutine stack. The `seq x y = ifzero x then y else y` hack in the old `aoc8.m` exists only to work around this, and it only works for integer accumulators.

### Code changes (in order of payoff/effort)

1. ✅ **`eval/eval.go` + `typecheck/typecheck.go`: native `seq :: * -> ** -> **`** — evaluates the first argument to WHNF, returns the second. Lets Miracula code force accumulators of any type and retires the `ifzero` idiom. (Script-level `seq` definitions in older files are shadowed by the builtin, which has identical semantics.)
2. ✅ **`stdenv.m`: strict left fold** — `foldl` built on `seq` (`foldl f z (x:xs) = seq z2 (foldl f z2 xs) where z2 = f z x`), keeping `sum`, `product`, etc. constant-space. Measured: `sum [1..1000000]` = 1.7 s, constant stack.
3. **`eval/eval.go`: shrink `Whnf` stack frames** — extract the large, rarely-hot cases (`AppNode` built-in dispatch, `ZF*`, sorting) into separate functions so the recursive frame drops from ~2.4 KB to a few hundred bytes; raises the safe forcing depth by ~10×.
4. **(Longer term) iterative thunk-chain forcing** — when `Whnf` encounters a chain of unevaluated `ThunkCell`s, push cells onto an explicit `[]*ThunkCell` work stack and update them from the innermost value outward, removing the depth limit entirely.

---

## 9. Environment Representation: Lexical Addressing & Allocation Reduction 🟠

Phase 6 removes the quadratic blow-up, but every variable reference still walks a linked list comparing strings, every argument allocates a one-binding `Env` frame, and every non-atomic argument allocates a `ThunkCell` + `ThunkNode`. The `cpu4.prof` GC share (~35–40 %) is this allocation pressure.

### Code changes

1. **Resolver pass (new file, e.g. `eval/resolve.go`)** run once per definition after desugaring:
   - Rewrite each `VarNode` into `LocalVarNode{Depth, Index int}` (position in the lexical frame stack) or `GlobalVarNode{Name string}` (direct `Globals` reference, no chain walk).
   - Collect multi-argument lambdas and `where`/`let` groups so each application/let allocates **one frame** (`Env{Vals []Node}`) instead of one frame per binding.
2. **`ast/ast.go`**: add the resolved node types and the slice-backed `Env` frame; keep the legacy path for the REPL's incremental typing until the resolver covers it.
3. **`eval/eval.go`**: `LocalVarNode` lookup becomes two array indexes; `GlobalVarNode` becomes one map read (or, better, a pre-resolved `*ThunkCell` pointer captured at resolve time — zero lookups).
4. **Thunk avoidance** (✅ first half implemented as `bindArg` in `eval.go`): an argument that is just a variable reference now passes its existing local binding through instead of allocating a fresh indirection thunk. Before this, every `loop acc (curr+1)`-style call wrapped `acc` in a new thunk, so long runs of iterations built an unbounded indirection chain — this is what crashed `aoc2.m` with a fatal stack overflow (~69 000-deep chains from its largest ranges). Still open: skip `ThunkCell` allocation for arithmetic operand subtrees whose evaluation is unconditional.

Expected effect: removes the remaining `Env.Lookup`/`memequal` cost entirely and cuts allocations per reduction roughly in half; estimated 2–4× on top of Phase 6 for `aoc8.m` (3.5 s → ~1 s).

---

## 10. Data-Structure Upgrades: Persistent Int-Keyed Maps & Real Vectors 🟡

1. **Persistent map (`ast.MapNode`)**: replace the copy-on-write `map[string]Node` with an immutable balanced tree or HAMT keyed by `int64` (fall back to string keys only for string maps):
   - `h_insert` becomes O(log n) with structural sharing instead of O(n) full copy.
   - Kills the `strconv.FormatInt` allocation on every `getMapKey` call.
   - Union-find in Phase 7 drops from O(E·N) worst case to O(E log N).
2. **Vector value type**: new `VecNode{Elems []ast.Node}` (plus `IntVecNode{[]int64}` fast path) with `to_vec`, `vec_get` (true O(1)), `vec_len`, `vec_set` (documented O(n) copy, or persistent via the same tree). Deprecate the current `list_get`/`list_set`, which re-convert the list on every call.
3. **`memoize`**: key integer arguments on the `int64` value directly; use `PrintNode` only for compound keys.

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
