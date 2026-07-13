# Miracula Improved: Proposed Enhancements for Advent of Code Scale

This document specifies a set of proposed language and runtime extensions to improve the performance, expressiveness, and scalability of Miracula for solving complex programming puzzles like Advent of Code.

---

## 1. High-Performance List & Array Built-ins

To solve the \(O(N)\) cost of list indexing and updates in recursive algorithms (such as Union-Find or array sweeps):

* **`list_get`**: `[num] -> num -> num`
  - *Description*: Retrieves the element at a specified index in a list of integers natively in \(O(1)\) time.
* **`list_set`**: `[num] -> num -> num -> [num]`
  - *Description*: Natively copies and updates the element at the specified index, avoiding deep recursive Miracula thunk construction.
* **`sort_by_key`**: `([num] -> num) -> [[num]] -> [[num]]`
  - *Description*: Natively sorts a list of lists (or tuples represented as lists) using a key projection function in \(O(N \log N)\) using Go's native sorting mechanisms.

---

## 2. Built-in Associative Maps & Sets

Purely functional list lookups (\(O(N)\)) choke on larger dataset inputs. While balanced trees can be written in pure Miracula, native maps and sets at the interpreter layer are significantly faster.

* **`h_lookup`**: `(map * **) -> * -> **`
  - *Description*: Returns the value for a key in a map, or crashes/returns a fallback if missing.
* **`h_insert`**: `(map * **) -> * -> ** -> (map * **)`
  - *Description*: Returns a new map with the key-value pair added using copy-on-write or structural sharing.
* **`member`**: `(set *) -> * -> bool`
  - *Description*: Performs an \(O(1)\) presence check in a native hash set.

---

## 3. String Splitting & Tokenization

Parsing inputs character-by-character is slow and complex. A native tokenizing primitive makes input parsing trivial.

* **`split`**: `[char] -> [char] -> [[char]]`
  - *Description*: Splits a string based on a set of delimiter characters.
  - *Example*: `split " ," "12, 34, 56"` returns `["12", "34", "56"]`.
* **`parse_ints`**: `[char] -> [num]`
  - *Description*: Natively extracts all integer values from a string.
* **`parse_3d_points`**: `[char] -> [(num, num, num)]`
  - *Description*: Natively parses lines of comma-separated coordinates directly into a list of 3D tuples.

---

## 4. Automatic Memoization

Dynamic programming and branching state-space search (DFS/BFS) require caching. A memoization wrapping primitive or attribute eliminates exponential state branching.

* **`memoize`**: `(* -> **) -> (* -> **)`
  - *Description*: Wraps a function so that evaluations are cached in an internal hash map using arguments as keys, returning the cached result upon repeat calls.

---

## 5. 64-bit Integers by Default

Advent of Code Part 2 puzzles frequently track numbers in the trillions. 

* *Requirement*: Ensure all native parser tokens and internal interpreter node representations utilize 64-bit signed integers (`int64` in Go) rather than standard 32-bit integers to prevent numeric overflow.

---

## 6. Efficient File I/O

* **`readfile`**: `[char] -> [char]`
  - *Description*: Natively reads the entire contents of a file into a character list (string) in a single operation.
