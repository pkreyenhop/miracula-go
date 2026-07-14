# Miracula (Miranda Interpreter in Go)

Miracula is a lightweight interpreter and interactive REPL for a lazy functional programming language inspired by Miranda. This repository contains the Go implementation, featuring lazy evaluation (call-by-need), pattern-matching desugaring, list primitives, lexical scoping, and an interactive REPL.

## Features

- **Lazy Evaluation (Call-by-Need):** Expressions are evaluated only when required using memoized thunks to avoid redundant computation. Includes infinite loop detection (`Blackhole` exception).
- **Static Type Inference (Hindley–Milner):** Every definition and REPL expression is type-checked before evaluation, with positioned error messages; no type declarations needed.
- **Lexical Closures (Lexical Scoping):** First-class environment-capturing closures that support lexical scope for nested curried functions, ensuring outer variable bindings are resolved correctly in recursive/nested calls. A post-typecheck resolver rewrites variable references to static coordinates, so lookups need no name comparison at run time.
- **Unbounded Evaluation Depth:** The evaluator runs on an explicit continuation stack (no Go-stack recursion for strict positions or thunk chains), so folds over millions of elements evaluate instead of overflowing.
- **High-Performance Native Builtins:** persistent AVL maps (`h_insert`/`h_lookup`/`h_lookup_def`), O(1) vectors (`to_vec`/`vec_get`/`vec_set`), native sorting (`sort_ints`/`sort_by`/`sort_edges`/`sort_pts`), input parsing (`split`/`parse_ints`/`read`/`lines`), `seq`, and `memoize` — see section 22 of the manual (`/m` in the REPL) for signatures and examples.
- **List Pattern Matching & Desugaring:** Allows defining functions through multiple equations with pattern matching on integers, characters, variables, and list patterns (`[]` and `(x:xs)` cons patterns) compiled into conditional decision trees.
- **Lazy List Ranges:** Dynamic sequence generators using `[e1..e2]` syntax (e.g., `[1..100]`), lazily evaluated step-by-step so that sequences are generated only as they are accessed.
- **Interactive REPL**: Provides a prompt (`miranda> `) to define variables/functions and evaluate expressions interactively.
  - **Enhanced Line Editing & Navigation**:
    - **Cursor Movement**: Left/Right arrow keys.
    - **History Navigation**: Up/Down arrow keys (retains typed draft line when scrolling back to current input).
    - **Go to Start/End**: `Ctrl-A` (or `Home`) and `Ctrl-E` (or `End`).
    - **Delete/Kill**: `Ctrl-K` (kill to end of line), `Backspace` (delete backward), and `Delete` (delete under cursor).
    - **Interrupt/Cancel**: `Ctrl-C` cancels the current text entry. If a computation is actively evaluating, `Ctrl-C` terminates the evaluation safely and returns control back to the REPL prompt without exiting.
    - **Autocomplete (Tab)**: Auto-completes keywords (`if`, `then`, `else`, `where`, `otherwise`, `mod`), built-in functions (`hd`, `tl`, `show`, `length`, etc.), and dynamically defined global variables/functions. Pressing `Tab` repeatedly cycles through matching candidates.
  - **Special REPL Commands**:
    - `/e [file.m]`: Open and edit the loaded script file (default `~/.script.m`) in `vi` (or `./mica`), reloading all definitions on exit.
    - `/m`: Open the combined language manual using the Unix `more` command.
    - `!COMMAND`: Execute `COMMAND` directly in the Unix shell (via `sh -c`).
    - `?FUNCTION`: Display the first line of `FUNCTION`'s definition.
    - `??FUNCTION`: Open the source file where `FUNCTION` is defined in the editor at the exact definition line.
    - `/?` or `/h`: Show the list of special REPL commands and their descriptions.
    - `/q`, `exit`, or `quit` (or `Ctrl-D` on empty line): Exit the REPL.
  - **Persistent Workspace**: All global variables and functions defined interactively inside the REPL are automatically saved to `~/.script.m`, which is loaded automatically at startup.

## Concrete Surface Syntax & AST
Miracula parses high-level surface syntax construct and desugars them into core primitives:
- **String Literals**: `"abc"` desugars to `Cons (Char 'a', Cons (Char 'b', Cons (Char 'c', Nil)))`.
- **Negation Prefix**: `-e` desugars to `Sub (Int 0, e)`.
- **List Length**: `#e` desugars to `App (Var "length", e)`.
- **Logical AND (`e1 & e2`)**: Desugars to `If (e1, e2, Int 0)`.
- **Logical OR (`e1 \/ e2`)**: Desugars to `If (e1, Int 1, e2)`.
- **Function Composition (`f . g`)**: Desugars to `Lam (cx, App (f, App (g, Var cx)))` with a fresh variable `cx`.
- **List Comprehensions (`[ e | q1; q2 ]`)**: Represented in the AST as `ZF (e, [q1, q2])` and evaluated dynamically via lazy generators.

## Directory Structure

The project follows an idiomatic Go layout:
- [ast/](file:///home/pkreyenhop/src/miracula-go/ast/ast.go): Defines AST types, pattern matching qualifiers, environment scope chains (`Env`), and execution errors.
- [cmd/miracula/](file:///home/pkreyenhop/src/miracula-go/cmd/miracula/main.go): Command-line entry point. Handles arguments, loads the stdenv, and spins up the TTY REPL.
- [eval/](file:///home/pkreyenhop/src/miracula-go/eval/eval.go): Weak Head Normal Form (WHNF) reducer running on an explicit continuation stack, native builtins, and the lexical-addressing resolver (`resolve.go`).
- [lexer/](file:///home/pkreyenhop/src/miracula-go/lexer/lexer.go): Text tokenizer, layout rules, and standard list formatting wrappers.
- [parser/](file:///home/pkreyenhop/src/miracula-go/parser/parser.go): Recursive descent parser and pattern-matching equations compiler/desugarer.
- [repl/](file:///home/pkreyenhop/src/miracula-go/repl/repl.go): Dynamic raw terminal line editor featuring tab-autocomplete, history navigation, and Emacs hotkeys.

## How to Build and Run

### Prerequisites
Make sure you have [Go](https://go.dev/) (version 1.18 or higher) installed on your system.

### Build
Compile the interpreter to produce a `miracula` binary:
```bash
go build -o miracula cmd/miracula/main.go
```

### Run
Launch the REPL by running the compiled executable:
```bash
./miracula [script_file]
```
If no script file argument is provided, the interpreter defaults to loading `~/.script.m`.

For example, to run the interactive REPL with the standard test suite:
```bash
./miracula test_miracula.m
```
Inside the REPL, type `main` to run all verification checks.

### Example Programs
The `examples/` directory contains a collection of small Miracula programs (classic Miranda examples such as `hanoi.m`, `queens.m`, and `primes.m`), plus complete Advent of Code 2025 solutions with their puzzle inputs under `examples/aoc25/`.

Run them from the repository root so `stdenv.m` and the input file paths resolve correctly. For example, to run the optimized Advent of Code Day 1 solution:
```bash
./miracula examples/aoc25/aoc.m
```
And enter `main` in the prompt to evaluate the results. `examples/aoc25/aoc_all.m` runs every day's solution in one go.
