# Miracula Improved: Detailed Implementation Plan

This document outlines the step-by-step technical implementation plan for extending the Miracula Go runtime and interpreter to support the high-performance primitives, 64-bit default integers, memoization, and data structures outlined in [MIRACULA_IMPROVED.md](file:///Users/pkreyenhop/src/miracula-go/MIRACULA_IMPROVED.md).

---

## 1. Upgrade to 64-bit Integers

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

## 2. Introduce Native Maps & Sets

To enable \(O(1)\) lookups and updates without mutating state globally in a lazy, pure functional environment, we will implement immutable Maps and Sets using structural copying or copy-on-write wrapper nodes.

### Code Changes:
1. **[ast/ast.go](file:///Users/pkreyenhop/src/miracula-go/ast/ast.go)**:
   - Define new AST nodes:
     ```go
     type MapNode struct {
         Map map[string]Node // Key-value store
     }
     type SetNode struct {
         Set map[string]bool // Member store
     }
     ```
   - Register their corresponding `isNode()` methods.
2. **[typecheck/typecheck.go](file:///Users/pkreyenhop/src/miracula-go/typecheck/typecheck.go)**:
   - Define type constructors for Maps and Sets in `typecheck`:
     - `TMap(ElemType)`
     - `TSet(ElemType)`
   - Register native helper functions in the type check environment:
     - `h_lookup :: TMap(k, v) -> k -> v`
     - `h_insert :: TMap(k, v) -> k -> v -> TMap(k, v)`
     - `member   :: TSet(k) -> k -> Bool`
3. **[eval/eval.go](file:///Users/pkreyenhop/src/miracula-go/eval/eval.go)**:
   - Implement evaluation handlers:
     - **`h_insert`**: Creates a shallow copy of the Go map inside `MapNode`, inserts the new key-value pair, and returns a new `MapNode`.
     - **`h_lookup`**: Resolves the key from the target `MapNode` and returns the associated node, throwing a runtime error if missing.
     - **`member`**: Natively checks presence in the Go map inside `SetNode` and returns a `BoolNode`.

---

## 3. String Splitting & Tokenization

* **`split`**: `[char] -> [char] -> [[char]]`
* **`parse_ints`**: `[char] -> [num]`

### Code Changes:
1. **[typecheck/typecheck.go](file:///Users/pkreyenhop/src/miracula-go/typecheck/typecheck.go)**:
   - Register helper type signatures:
     - `split :: List(TChar) -> List(TChar) -> List(List(TChar))`
     - `parse_ints :: List(TChar) -> List(TInt)`
2. **[eval/eval.go](file:///Users/pkreyenhop/src/miracula-go/eval/eval.go)**:
   - Implement **`split`**: Convert the delimiters and input character lists to Go strings. Use `strings.FieldsFunc` or `strings.Split` in Go to split the input string, then convert the resulting slices back into Miracula lists of strings (character lists).
   - Implement **`parse_ints`**: Natively scan the Go string representation of the character list for numeric tokens using standard regular expressions (`-?\d+`), parse them into `int64`, and wrap them in a Miracula list.

---

## 4. Native List Indexing & Updates

* **`list_get`**: `[num] -> num -> num`
* **`list_set`**: `[num] -> num -> num -> [num]`

### Code Changes:
1. **[typecheck/typecheck.go](file:///Users/pkreyenhop/src/miracula-go/typecheck/typecheck.go)**:
   - Register signatures:
     - `list_get :: List(TInt) -> TInt -> TInt`
     - `list_set :: List(TInt) -> TInt -> TInt -> List(TInt)`
2. **[eval/eval.go](file:///Users/pkreyenhop/src/miracula-go/eval/eval.go)**:
   - Implement **`list_get`**: Convert the Miracula list into a Go slice of integers. Retrieve the element at index `k` in \(O(1)\) time.
   - Implement **`list_set`**: Convert the Miracula list into a Go slice, copy the slice, update the element at index `k` in \(O(1)\) time, and convert it back into a Miracula list representation.

---

## 5. Automatic Function Memoization

### Code Changes:
1. **[ast/ast.go](file:///Users/pkreyenhop/src/miracula-go/ast/ast.go)**:
   - Define a memoization wrapper node:
     ```go
     type MemoizeNode struct {
         Func  Node
         Cache map[string]Node // Caches evaluated results keyed by argument string representations
     }
     ```
2. **[typecheck/typecheck.go](file:///Users/pkreyenhop/src/miracula-go/typecheck/typecheck.go)**:
   - Register the signature:
     - `memoize :: (a -> b) -> (a -> b)`
3. **[eval/eval.go](file:///Users/pkreyenhop/src/miracula-go/eval/eval.go)**:
   - Implement the evaluation of `MemoizeNode`.
   - When a function wrapped inside `MemoizeNode` is called, serialize its argument node using `PrintNode` to obtain a unique key.
   - Check the cache map. If the result is present, return it immediately.
   - If not, evaluate the function application, store the evaluated WHNF node back in the cache map, and return the result.
