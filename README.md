# Miracula (Miranda Interpreter in Go)

Miracula is a lightweight interpreter and interactive REPL for a lazy functional programming language inspired by Miranda. This repository contains the Go implementation, featuring lazy evaluation (call-by-need), pattern-matching desugaring, list primitives, lexical scoping, and an interactive REPL.

## Features

- **Lazy Evaluation (Call-by-Need):** Expressions are evaluated only when required using memoized thunks to avoid redundant computation. Includes infinite loop detection (`Blackhole` exception).
- **Lexical Closures (Lexical Scoping):** First-class environment-capturing closures that support lexical scope for nested curried functions, ensuring outer variable bindings are resolved correctly in recursive/nested calls.
- **List Pattern Matching & Desugaring:** Allows defining functions through multiple equations with pattern matching on integers, characters, variables, and list patterns (`[]` and `(x:xs)` cons patterns) compiled into conditional decision trees.
- **Lazy List Ranges:** Dynamic sequence generators using `[e1..e2]` syntax (e.g., `[1..100]`), lazily evaluated step-by-step so that sequences are generated only as they are accessed.
- **Interactive REPL:** Provides a prompt (`miranda> `) to define variables/functions and evaluate expressions interactively.
  - **Enhanced Line Editing**: Basic editing via standard REPL input.
  - `/e` command: Open and edit the loaded script file, reloading all definitions on exit.
  - `/q` command: Exit the REPL.

## Concrete Surface Syntax & AST
Miracula parses high-level surface syntax construct and desugars them into core primitives:
- **String Literals**: `"abc"` desugars to `Cons (Char 'a', Cons (Char 'b', Cons (Char 'c', Nil)))`.
- **Negation Prefix**: `-e` desugars to `Sub (Int 0, e)`.
- **List Length**: `#e` desugars to `App (Var "length", e)`.
- **Logical AND (`e1 & e2`)**: Desugars to `If (e1, e2, Int 0)`.
- **Logical OR (`e1 \/ e2`)**: Desugars to `If (e1, Int 1, e2)`.
- **Function Composition (`f . g`)**: Desugars to `Lam (cx, App (f, App (g, Var cx)))` with a fresh variable `cx`.
- **List Comprehensions (`[ e | q1; q2 ]`)**: Represented in the AST as `ZF (e, [q1, q2])` and evaluated dynamically via lazy generators.

## How to Build and Run

### Prerequisites
Make sure you have [Go](https://go.dev/) (version 1.18 or higher) installed on your system.

### Build
Compile the interpreter to produce a `miracula` binary:
```bash
go build -o miracula main.go
```

### Run
Launch the REPL by running the compiled executable:
```bash
./miracula [script_file]
```
If no script file argument is provided, the interpreter defaults to loading `script.m` if present.

For example, to run the interactive REPL with the standard test suite:
```bash
./miracula test_miracula.m
```
Inside the REPL, type `main` to run all verification checks.

### Running Advent of Code Solution
You can run the optimized Advent of Code Day 1 solution using:
```bash
./miracula aoc.m
```
And enter `main` in the prompt to evaluate the results.
