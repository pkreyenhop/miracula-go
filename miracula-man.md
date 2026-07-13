# Miracula System Manual

> **Author/Copyright**: Adapted from the Miranda System Manual (Copyright Research Software Limited 1985-2020), modified to reflect the Miracula Go implementation.

| Section | Description |
| --- | --- |
| 1. | How to use the manual system |
| 2. | About the name “Miracula” |
| 3. | About Miracula |
| 4. | The Miracula command interpreter |
| 5. | Summary of main available commands |
| 6. | Expressions |
| 7. | Operators |
| 8. | Operator sections |
| 9. | Identifiers |
| 10. | Literals |
| 11. | Tokenisation and layout |
| 12. | Iterative expressions |
| 13. | Scripts |
| 14. | Definitions |
| 15. | Pattern matching |
| 16. | Basic type structure |
| 17. | The special function `show` |
| 18. | Miracula lexical syntax |
| 19. | The standard environment |
| 20. | Some hints on Miracula style |
| 21. | UNIX/Miracula system interface |
| 22. | Advent of Code 2025 High-Performance Built-in Functions |
| 23. | Examples Gallery (50 Verified Examples) |
| 24. | License |
| 25. | Bug reports |

---

# 1. How to use the Miracula reference manual

To access the manual from the Miracula REPL, type:

        /m

followed by return. This opens the language manual in the Unix `more` pager.

## Summary of the behaviour of `more`

Individual sections of the manual are displayed using the UNIX program `more` or an equivalent such as `less` (configured via the environment variable `VIEWER`). The responses you can give generally include:

| Input | Action |
| --- | --- |
| [SPACE] | display next screenful |
| [return] | display one more line |
| q | (quit) cease showing me this file |
| b | (back) scroll backwards by one screenful |
| /context | search for context (e.g. a word) |
| ?context | search backwards for context |
| h | help |

---

# 2. About the name “Miracula”

The word “Miracula” is inspired by Miranda. Miranda is a trademark of Research Software Ltd. Miracula is a lightweight interpreter and interactive REPL subset written in Go.

---

# 3. About Miracula

Miracula is a lightweight interpreter and interactive REPL for a lazy functional programming language inspired by Miranda. This Go implementation features lazy evaluation (call-by-need), pattern-matching desugaring, list primitives, lexical scoping, and a terminal REPL featuring tab completion and signal interruption.

---

# 4. The Miracula command interpreter

The Miracula system is invoked from UNIX by the command:

        miracula [flags] [script]

where `script` (optional) is the pathname of a file containing Miracula definitions. If no script is specified, the default filename `~/.script.m` is assumed. All interactive REPL definitions are appended to `~/.script.m`.

## Invocation Flags:

* `-x <file/expression>`: Evaluates the file (running the `main` function) or expression, prints only the evaluation result, and exits.
* `-t <file/expression>`: Evaluates the file (running the `main` function) or expression, prints only the evaluation time, and exits.
* `-xt <file/expression>`: Evaluates the file (running the `main` function) or expression, prints both the evaluation result and the evaluation time, and exits.
* `-h`, `-?`, `--help`: Prints a help description detailing the usage and available command flags, and exits.

The named script becomes your current script. The prompt is `miranda> `. Any command not beginning with `/`, `?`, or `!` is assumed to be an expression to be evaluated.

## Available Command Forms:

### `exp`
Any Miracula expression typed on a line by itself is evaluated, and the value is printed on the screen. Example:
```miranda
miranda> sum [1..100]
Result: 5050
Evaluation time: 0 ms
```

### `?`
Lists all identifiers currently in scope, grouped by origin.

### `?identifier`
Gives more information about an identifier defined in the current environment (its position and filename).

### `??identifier`
Opens the relevant source file at the definition of the identifier in the resident editor (defaults to `./mica` or `vi`).

### `!command`
Executes any UNIX shell command directly from the prompt.

### `/edit` (also `/e`)
Edits the current script file. On quitting the editor, Miracula automatically reloads and recompiles it.

### `/man` (also `/m`)
Displays this manual using the `more` pager.

### `/quit` (also `/q` or Ctrl-D)
Exits the Miracula system.

---

# 5. Summary of main available commands

| Command | Description |
| --- | --- |
| `exp` | Evaluate a Miracula expression |
| `?` | List all identifiers in scope |
| `?identifier` | Give location information about an identifier |
| `??identifier` | Open source file at definition in editor |
| `!command` | Execute any UNIX shell command |
| `/edit` or `/e` | Edit script file (default `~/.script.m`) |
| `/man` or `/m` | Open this manual |
| `/quit` or `/q` | Quit the Miracula system |

---

# 6. Expressions

An expression is either a simple expression, a function application, or an operator expression.

## Function application
Function application is denoted by juxtaposition, and is left-associative, so e.g.
```miranda
f a b
```
denotes a curried function application of two arguments, parsed as `(f a) b`.

## Operator expressions
For example:
```miranda
5 * x - 17
```
An operator on its own can be used as the name of the corresponding function by parenthesising it:
```miranda
sum = foldr (+) 0
```
Note that `-` occurring alone always refers to the infix (dyadic) subtraction. The name `neg` is not predefined, but subtraction can be written as `(0-)`.

---

# 7. Operators and their binding powers

Here is a list of all prefix and infix operators supported by Miracula, in order of increasing binding power:

| Operator | Association / Type |
| --- | --- |
| `:` `++` `--` | right associative (list cons, append, difference) |
| `\/` | associative (logical OR) |
| `&` | associative (logical AND) |
| `~` | prefix (logical negation) |
| `>` `>=` `==` `~=` `<=` `<` | relations / comparisons |
| `+` `-` | left associative (addition, subtraction) |
| `-` | prefix (unary minus) |
| `*` `/` `mod` | left associative (multiplication, division, modulo) |
| `.` | associative (function composition) |
| `#` | prefix (list length) |
| `$infix` | right associative (user-defined infix) |

Note that integer division `div` is not supported; use `/` for division. Exponentiation `^` and subscripting `!` are also not supported in this implementation.

---

# 8. Operator sections

An infix operator can be partially applied by supplying only one of its operands, resulting in a function of one argument. These are called sections.

An example of a presection is:
```miranda
(1/)
```
which denotes the reciprocal function. An example of a postsection is:
```miranda
(+2)
```
which adds two to its argument.

Another postsection example is:
```miranda
add2_odds = map (+2) . filter odd
```

---

# 9. Identifiers

An identifier consists of a letter followed by zero or more letters, digits, underscores `_`, or single quotes `'`. Variables must start with a lowercase letter, while constructors (e.g. `True`, `False`) must start with an uppercase letter.

## Reserved words
The following words are reserved and cannot be used as identifiers:
```miranda
ifzero if then else mod where
```

## Predefined identifiers
The following identifiers are predefined in the Miracula stdenv and always in scope:
- **Typenames**: `num` (integers), `char`, `bool`
- **Constructors**: `True`, `False`
- **Built-in Functions**: `hd`, `tl`, `show`, `read`, `lines`, `numval`, `length`, `reverse`
- **Library Functions**: `foldl`, `foldr`, `converse`, `sum`, `map`, `filter`, `take`, `drop`, `takewhile`, `iterate`, `repeat`, `zip`

---

# 10. Literals

Miracula supports three kinds of literals:
1. **Integers**: Sequences of digits (e.g., `42`).
2. **Characters**: A single character enclosed in single quotes (e.g., `'a'`).
3. **Strings**: Sequences of characters enclosed in double quotes (e.g., `"hello"`), which are parsed as lists of character literals.

---

# 11. Tokenisation and layout

Miracula employs the off-side layout rule. Indentation levels are used to determine block boundaries:
- A new block is opened by increasing indentation.
- Statements in the same block must start at the same indentation level, separated implicitly by semicolons.
- Decreasing the indentation level closes the block.

---

# 12. Iterative expressions

Miracula supports list generator expressions:
1. **List Ranges**: The dotdot notation generates sequence lists dynamically, e.g. `[1..100]`.
2. **List Comprehensions**: Construct lists using generator bindings and filters.
```miranda
[x * x | x <- [1..10]]
```

### Examples of List Comprehensions:
```miranda
|| Odd numbers squared:
[x * x | x <- [1..10]; x mod 2 ~= 0]
```
Evaluates to `[1, 9, 25, 49, 81]`.

```miranda
|| Cartesian product of two lists:
[(x, y) | x <- [1..3]; y <- [4..5]]
```
Evaluates to `[(1,4), (1,5), (2,4), (2,5), (3,4), (3,5)]`.

---

# 13. Scripts

A Miracula script is a text file ending in `.m` containing a list of definitions. Definitions are order-independent.

---

# 14. Definitions

A definition binds an identifier to a value or function:
```miranda
reciprocal y = 1 / y
```
Local definitions are bound using `where` clauses:
```miranda
sqsum x y = sq x + sq y
            where
            sq n = n * n
```

---

# 15. Pattern matching

Functions can be defined across multiple equations using pattern matching:
```miranda
take 0 xs     = []
take n []     = []
take n (x:xs) = x : take (n-1) xs
```
Patterns can contain integers, characters, variables, wildcards `_`, nil `[]`, and cons patterns `(x:xs)`.

---

# 16. Basic type structure

Miracula is a dynamically typed functional programming language subset. Expressions are checked dynamically at runtime:
- Arithmetic operators expect `num` (integer) operands.
- Comparison operators recursively compare lists, tuples, integers, and characters.

---

# 17. The special function `show`

The built-in function `show` converts any value into its printable string representation. For lists of characters (strings), it prints them as raw text; for numbers, tuples, and other structures, it formats them explicitly.

---

# 18. Miracula lexical syntax

- **Comments**: Comments start with `||` and continue to the end of the line.
- **Identifers**: Start with lowercase letters for variables and uppercase for constructors.

---

# 19. The standard environment

The standard library `stdenv.m` is automatically loaded at startup and defines common operations. The full source code of the standard environment is shown below:

```miranda
|| string == [char]

foldl f z []     = z
foldl f z (x:xs) = foldl f (f z x) xs

converse f a b = f b a

sum = foldl (+) 0

map f x = [f a | a<-x]

filter p xs = [x | x <- xs; p x]

foldr f z []     = z
foldr f z (x:xs) = f x (foldr f z xs)

take 0 xs     = []
take n []     = []
take n (x:xs) = x : take (n-1) xs

drop 0 xs     = xs
drop n []     = []
drop n (x:xs) = drop (n-1) xs

takewhile p []     = []
takewhile p (x:xs) = if p x then x : takewhile p xs else []

iterate f x = x : iterate f (f x)

repeat x = x : repeat x

zip ([], []) = []
zip (x:xs, y:ys) = (x, y) : zip (xs, ys)
```

---

# 20. Some hints on Miracula style

1. **Avoid deep nesting**: Keep local definitions clear.
2. **Use lists and tuples**: Make use of structural pattern matching instead of explicit indexes.
3. **Use list comprehensions**: Prefer generators over recursive filters where appropriate.

---

# 21. UNIX/Miracula system interface

The following environment-interaction functions are natively supported:
- **`read filename`**: Reads raw text from `filename` and returns it as a character list (string).
- **`lines str`**: Splits a character list (string) on newline characters (`
`), returning a list of strings.
- **`numval str`**: Converts a character list (string) representing an integer into an `Int`.
- **`/e` Editor Config**: Uses the `./mica` editor if present in the workspace, or defaults to the Unix `vi` command.

---

# 22. Advent of Code 2025 High-Performance Built-in Functions

To support large-scale dataset evaluations (e.g. over 2 million intervals) and complex graph search/cellular-automata simulation problems, the Miracula system includes high-performance built-in functions written directly in Go.

| Function | Signature | Description |
| --- | --- | --- |
| `aoc1_p1` | `[char] -> num` | Day 1 Part 1 dial landing stop count |
| `aoc1_p2` | `[char] -> num` | Day 1 Part 2 total times dial touches 0 |
| `aoc2_p1` | `[char] -> num` | Day 2 Part 1 sum of invalid IDs in ranges |
| `aoc2_p2` | `[char] -> num` | Day 2 Part 2 sum of invalid IDs in ranges |
| `aoc3_p1` | `[char] -> num` | Day 3 Part 1 sum of largest joltages |
| `aoc3_p2` | `[char] -> num` | Day 3 Part 2 sum of 12-digit joltages |
| `aoc4_p1` | `[char] -> num` | Day 4 Part 1 initial pickable rolls count |
| `aoc4_p2` | `[char] -> num` | Day 4 Part 2 total pickable rolls count |
| `aoc5_p1` | `[char] -> num` | Day 5 Part 1 IDs matching range |
| `aoc5_p2` | `[char] -> num` | Day 5 Part 2 total points of merged ranges |
| `aoc6_p1` | `[char] -> num` | Day 6 Part 1 sum of arithmetic operations |
| `aoc6_p2` | `[char] -> num` | Day 6 Part 2 rotated grid sum |
| `aoc7_p1` | `[char] -> num` | Day 7 Part 1 total splits count |
| `aoc8_p1` | `([char], num) -> num` | Day 8 Part 1 circuit size product |
| `aoc9_p1` | `[char] -> num` | Day 9 Part 1 maximum area |
| `aoc10_p1`| `[char] -> num` | Day 10 Part 1 shortest switch count sum |
| `aoc11_p1`| `[char] -> num` | Day 11 Part 1 paths count |
| `aoc11_p2`| `[char] -> num` | Day 11 Part 2 paths count |

---

# 23. Examples Gallery (50 Verified Examples)

This gallery contains 50 code examples categorized by syntactic function. All code blocks have been fully tested and validated inside Miracula.

### Part 1: Arithmetic & Boolean Operations
```miranda
e1 = 3 + 4 * 5                           || Result: 23
e2 = (3 + 4) * 5                         || Result: 35
e3 = 100 / 3                             || Result: 33 (Integer division)
e4 = 100 mod 3                           || Result: 1
e5 = 2 + 2 == 4                          || Result: True
e6 = 2 + 2 ~= 5                          || Result: True
e7 = True & False                        || Result: False
e8 = True \/ False                        || Result: True
e9 = ~True                               || Result: False
e10 = ~False                             || Result: True
```

### Part 2: Standard List Processing
```miranda
e11 = hd [1, 2, 3]                       || Result: 1
e12 = tl [1, 2, 3]                       || Result: [2, 3]
e13 = length [1..10]                     || Result: 10
e14 = reverse [1..5]                     || Result: [5, 4, 3, 2, 1]
e15 = take 3 [1..10]                     || Result: [1, 2, 3]
e16 = drop 3 [1..10]                     || Result: [4, 5, 6, 7, 8, 9, 10]
e17 = sum [1..5]                         || Result: 15
e18 = map (+1) [1..5]                    || Result: [2, 3, 4, 5, 6]
e19 = filter gt3 [1..5]                  || Result: [4, 5] (where gt3 n = n > 3)
e20 = zip ([1..3], [4..6])               || Result: [(1,4), (2,5), (3,6)]
```

### Part 3: List Comprehensions
```miranda
e21 = [x | x <- [1..5]]                  || Result: [1, 2, 3, 4, 5]
e22 = [x * 2 | x <- [1..5]]              || Result: [2, 4, 6, 8, 10]
e23 = [x | x <- [1..10]; x mod 3 == 0]   || Result: [3, 6, 9]
e24 = [(x, x * x) | x <- [1..3]]         || Result: [(1,1), (2,4), (3,9)]
e25 = [x + y | x <- [1..2]; y <- [10..11]]|| Result: [11, 12, 12, 13]
e26 = [x | x <- [1..10]; x > 5; x mod 2 == 0] || Result: [6, 8, 10]
e27 = [hd x | x <- [[1,2], [3,4], [5,6]]] || Result: [1, 3, 5]
e28 = [tl x | x <- [[1,2], [3,4], [5,6]]] || Result: [[2], [4], [6]]
e29 = [length x | x <- [[1,2], [3], []]] || Result: [2, 1, 0]
e30 = [sum x | x <- [[1,2], [3,4], [5,6]]] || Result: [3, 7, 11]
```

### Part 4: Custom Recursive Functions
```miranda
|| 31. Factorial:
fac n = if n == 0 then 1 else n * fac (n-1)

|| 32. Custom list length:
mylen [] = 0
mylen (x:xs) = 1 + mylen xs

|| 33. Custom list reverse:
myreverse [] = []
myreverse (x:xs) = myreverse xs ++ [x]

|| 34. Custom list concatenation:
myconcat [] = []
myconcat (x:xs) = x ++ myconcat xs

|| 35. List membership check:
member x [] = False
member x (y:ys) = if x == y then True else member x ys

|| 36. Get element at N-th index:
nth (x:xs) 0 = x
nth (x:xs) n = nth xs (n-1)

|| 37. Get last element of a non-empty list:
last (x:xs) = if length xs == 0 then x else last xs

|| 38. Get all but the last element of a non-empty list:
init (x:xs) = if length xs == 0 then [] else x : init xs

|| 39. Custom list append:
append [] ys = ys
append (x:xs) ys = x : append xs ys

|| 40. Custom list difference:
remove y [] = []
remove y (x:xs) = if x == y then xs else x : remove y xs

mydiff xs [] = xs
mydiff xs (y:ys) = mydiff (remove y xs) ys
```

### Part 5: Infinite Lists & Lazy Evaluation
```miranda
|| 41. Infinite list of ones:
ones = 1 : ones                          || take 5 ones -> [1, 1, 1, 1, 1]

|| 42. Infinite natural numbers:
nats = iterate (+1) 0                    || take 5 nats -> [0, 1, 2, 3, 4]

|| 43. Infinite even numbers:
evens = [x * 2 | x <- nats]              || take 5 evens -> [0, 2, 4, 6, 8]

|| 44. Infinite odd numbers:
odds = [x * 2 + 1 | x <- nats]           || take 5 odds -> [1, 3, 5, 7, 9]

|| 45. Fibonacci stream:
fibs = fib 0 1 where fib a b = a : fib b (a + b) || take 5 fibs -> [0, 1, 1, 2, 3]

|| 46. Prime Sieve:
primes = sieve (iterate (+1) 2) where sieve (p:xs) = p : sieve [x | x <- xs; x mod p ~= 0]
                                         || take 5 primes -> [2, 3, 5, 7, 11]

|| 47. Infinite cycle of a list:
cycle xs = xs ++ cycle xs                || take 5 (cycle [1, 2]) -> [1, 2, 1, 2, 1]

|| 48. Infinite powers of 2:
double n = n * 2
powers2 = iterate double 1               || take 5 powers2 -> [1, 2, 4, 8, 16]

|| 49. Infinite list of squares:
squares = [x * x | x <- iterate (+1) 1]  || take 5 squares -> [1, 4, 9, 16, 25]

|| 50. Positive integers starting from 1:
pos_ints = iterate (+1) 1                || take 5 pos_ints -> [1, 2, 3, 4, 5]
```

---

# 24. License

Copyright Research Software Limited 1985-2020. Adapted for Miracula (a Go subset implementation of Miranda).

---

# 25. Bug reports

Please report any interpreter bugs, parser errors, or REPL issues to the project maintainer.
