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
| 22. | High-performance built-in functions |
| 23. | Examples Gallery (50 Verified Examples) |
| 24. | Miracula for Miranda users |
| 25. | Miracula for Haskell users |
| 26. | Miracula for Admiran users |
| 27. | License |
| 28. | Bug reports |

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
Note that `-` occurring alone always refers to the infix (dyadic) subtraction. The name `neg` is not predefined; write negation as `(- x)` in expressions or `\x. 0 - x` as a function (presections like `(0-)` are not supported — see section 8).

## Lambda abstractions
Anonymous functions are written with a backslash and a dot, and can be nested for multiple arguments:
```miranda
miranda> (\x. x * x + 1) 6
Result: 37
miranda> sort_by (\a. \b. b - a) [3,1,2]
Result: [3,2,1]
```
(Note the dot: Haskell-style `\x -> e` is not accepted.)

A lambda's parameter may be a **tuple or cons pattern**, destructured on entry:
```miranda
miranda> map (\(a, b). a + b) [(1,2),(3,4)]
Result: [3,7]
miranda> sort_by (\(k1, v1). \(k2, v2). k1 - k2) [(3,"c"),(1,"a")]
Result: [(1,"a"),(3,"c")]
```

## The pipe operator
`x |> f` applies `f` to `x`. It is left-associative and binds loosest of all operators, so data flows left to right through a chain — often more readable than nested application:
```miranda
miranda> "peter" |> reverse |> hd
Result: 'r'
miranda> [1,2,3] |> map (\x. x * x) |> sum
Result: 14
miranda> "a,b" |> split "," |> length
Result: 2
```
Because it binds loosest, `1 + 2 |> (+10)` reads `(1 + 2) |> (+10)` = `13`. A lambda used directly as a pipe target must be parenthesised.

## Conditional expressions
`if` requires both branches and an expression condition; `ifzero` is a specialised form testing an integer against zero:
```miranda
miranda> if 3 > 2 then "yes" else "no"
Result:
yes
miranda> ifzero (3 - 3) then 1 else 2
Result: 1
```

## Local bindings: `let … in`
A `let` expression introduces local bindings for its body, body-first (the
complement of `where`, which comes body-last). Bindings are separated by `;` and
must fit on one logical line (use `where` for multi-line blocks); like `where`
they are mutually recursive and may bind patterns (see section 14):
```miranda
miranda> let x = 6; y = x * 2 in x + y
Result: 18
miranda> let (q, r) = (17 / 5, 17 mod 5) in q * 100 + r
Result: 302
```

---

# 7. Operators and their binding powers

Here is the complete list of prefix and infix operators supported by Miracula, in order of increasing binding power (this table matches the parser grammar exactly):

| Operator | Association / Type |
| --- | --- |
| `|>` | left associative (pipe: `x |> f` is the application `f x`) |
| `\/` | right associative (logical OR, short-circuit) |
| `&` | right associative (logical AND, short-circuit) |
| `~` | prefix (logical negation; binds looser than comparisons, so `~ a == b` reads `~(a == b)`) |
| `:` | right associative (list cons) |
| `++` `--` | right associative (list append, list difference) |
| `==` `~=` `<` `<=` `>` `>=` | comparisons (non-associative: `a < b < c` is a syntax error) |
| `+` `-` | left associative (addition, subtraction) |
| `*` `/` `mod` | left associative (multiplication, integer division, modulo — all one level) |
| `.` | right associative (function composition) |
| *juxtaposition* | left associative (function application) |
| `#` | prefix (list length; applies to one atom: `#xs + 1` is `(#xs) + 1`) |
| `-` | prefix (unary minus at atom level, e.g. `-x * y` is `(0-x) * y`) |

Examples (all verified):
```miranda
miranda> ~False & True
Result: True
miranda> 1 : [2] ++ [3]
Result: [1,2,3]
miranda> #[1,2,3] + 1
Result: 4
```

Additional notes:
- `!=` is accepted as an alias for `~=`.
- The ordering operators `<`, `<=`, `>`, `>=` are **polymorphic and structural**, like `==`: the two operands must have the same type, and any of `num`, `char`, `bool` (with `False < True`), or lists and tuples built from them may be compared. Lists compare lexicographically (`[]` precedes any non-empty list) and tuples element-wise:
```miranda
miranda> 'a' < 'b'
Result: True
miranda> "abc" < "abd"
Result: True
miranda> [1,2,3] < [1,3]
Result: True
```
This makes sorting strings straightforward with `sort_by`.
- Integer division truncates toward negative infinity (`100 / 3` is `33`); there is no separate `div`. Exponentiation `^` and subscripting `!` are not supported (for subscripting use `vec_get` on a vector).
- Division or modulo by zero raises a runtime error.
- `->` is tokenised but not part of any construct — lambdas use a dot (`\x. e`), not an arrow.
- Characters with no meaning to the language (`$`, `%`, `^`, `@`, a lone `!`, braces typed directly, …) are rejected with a positioned parse error:
```
miranda> a $ b
<stdin>:1:3: unrecognised character "$"
```
Inside string literals and `||` comments any character is of course fine.

---

# 8. Operator sections

A binary operator can be used as a function or partially applied by parenthesising it. Two forms are supported for every binary operator:

- **`(op)`** — the bare two-argument operator, e.g. `foldl (+) 0 xs`, `sort_by (-) …`, `(:) 0 [1]` → `[0,1]`.
- **`(op e)`** — a *right section* `\x. x op e`, e.g. `map (* 2) [1..3]` → `[2,4,6]`, `filter (> 3) xs`, `map (mod 2) xs`, `(: [2,3]) 1` → `[1,2,3]`.

The operators that section are `+ * / mod == ~= < > <= >= ++ -- : & \/`.

```miranda
miranda> (map (* 2) [1,2,3], filter (> 3) [1,2,3,4,5], foldl (*) 1 [1..4])
Result: ([2,4,6],[4,5],24)
```

Two caveats:
- `-` is special: `(-)` is two-argument subtraction, but `(- e)` is **unary minus** (so `(-2)` is the number −2), not a section. For "subtract from", write a lambda: `\x. 10 - x`.
- Only *right* sections `(op e)` exist; there is no *left* section `(e op)`. Since `+`, `*` etc. are commutative that rarely matters; otherwise use a lambda (`\x. e op x`).

---

# 9. Identifiers

An identifier consists of a letter followed by zero or more letters, digits, underscores `_`, or single quotes `'`. Variables must start with a lowercase letter, while constructors (e.g. `True`, `False`) must start with an uppercase letter.

## Reserved words
The following words are reserved and cannot be used as identifiers:
```miranda
ifzero if then else mod where let in
```

## Predefined identifiers
The following identifiers are predefined in the Miracula stdenv and always in scope:
- **Typenames**: `num` (64-bit integers), `char`, `bool`
- **Constructors**: `True`, `False`
- **Built-in Functions** (implemented natively in Go; see sections 21 and 22):
  - core: `hd`, `tl`, `show`, `read`, `lines`, `numval`, `length`, `reverse`, `seq`
  - characters: `ord` (char → code), `chr` (code → char)
  - string processing: `split`, `parse_ints`
  - maps and sets: `empty_map`, `h_insert`, `h_lookup`, `h_lookup_def`, `empty_set`, `member`
  - vectors: `to_vec`, `vec_get`, `vec_set`, `vec_len`, `vec_to_list`
  - priority queues: `pq_empty`, `pq_push`, `pq_pop`, `pq_null`
  - bitwise: `xor`, `band`, `bor`, `shl`, `shr`
  - sorting: `sort_ints`, `sort_by`, `sort_edges`, `sort_pts`
  - other: `memoize`, `memofix`, `list_get`, `list_set`
- **Library Functions**: `foldl`, `foldr`, `converse`, `sum`, `map`, `filter`, `take`, `drop`, `takewhile`, `iterate`, `repeat`, `zip`

A local binding — a lambda parameter, a `where` binding, or a comprehension generator — **shadows** a built-in of the same name within its scope, so naming a helper `split` or `member` is safe. At the top level the native built-in names above remain reserved: a script definition of the same name does not override the built-in.

```miranda
demo = a + b
       where
       member = 10        || a local binding named like the built-in `member`
       a = member * 2     || refers to the local, not the built-in
       b = member + 1

miranda> demo
Result: 31
```

---

# 10. Literals

Miracula supports three kinds of literals:
1. **Integers**: Sequences of digits (e.g., `42`). Integers are 64-bit signed throughout the lexer, evaluator, and native parsers.
2. **Characters**: A single character enclosed in single quotes (e.g., `'a'`).
3. **Strings**: Sequences of characters enclosed in double quotes (e.g., `"hello"`), which are parsed as lists of character literals.

---

# 11. Tokenisation and layout

A script or expression is composed of *tokens* — identifiers, literals, keywords, and delimiter symbols — separated by *layout*: spaces, tabs, newlines, and `||` comments.

Layout is ignored except in two respects.

**1. Token separation.** At least one layout character must separate two tokens that would otherwise merge into one. `f 19` is an application; `f19` is a single identifier, because digits are legal identifier characters.

**2. The offside rule** (Landin). Every token of a definition's right-hand side must lie directly below or to the right of the column where the right-hand side begins. A token further left is "offside" and ends the definition — which is why scripts need no explicit terminators:

```miranda
f x = y + z
      where
      y = (x + 1) * (x - 1)
      z = y * 2
g r = f (r + 1)

miranda> g 2
Result: 24
```

The `g` in column 1 is offside with respect to `f`'s right-hand side, so it starts a new top-level definition — while `y` and `z`, indented under the `where`, are local to `f`. Nested `where` clauses work the same way: each deeper block is indented further (section 14).

Within one line, `where` bindings may instead be separated by explicit semicolons:

```miranda
miranda> a + b where a = 1; b = 2
Result: 3
```

[Reference: P. J. Landin, "The Next 700 Programming Languages", CACM 9(3), 1966.]

---

# 12. Iterative expressions

Miracula supports list generator expressions:
1. **List Ranges**: The dotdot notation generates sequence lists lazily. `[1..100]` is the finite range; `[1..]` (no upper bound) is an infinite lazy list of consecutive integers — safe to build and process as long as only a finite prefix is demanded. A **stepped** range `[a, b .. c]` gives the arithmetic sequence with increment `b - a`, and `[a, b ..]` is its infinite form:
```miranda
miranda> ([1, 3 .. 9], [10, 8 .. 0], take 4 [0, 5 ..])
Result: ([1,3,5,7,9],[10,8,6,4,2,0],[0,5,10,15])
```
```miranda
miranda> hd [1..]
Result: 1
miranda> take 5 [1..]
Result: [1,2,3,4,5]
miranda> takewhile (\x. x * x < 50) [1..]
Result: [1,2,3,4,5,6,7]
miranda> hd [ x | x <- [1..]; x mod 7 == 0 ]
Result: 7
```
Evaluating an infinite list *in full* (printing `[1..]` itself, or `sum [1..]`, `#[1..]`) of course never finishes — interrupt with Ctrl-C.
2. **List Comprehensions**: Construct lists using generator bindings and filters. Generators may bind patterns — an element that fails the pattern is skipped:
```miranda
miranda> [ k + v | (k, v) <- zip ([1,2], [10,20]) ]
Result: [11,22]
```

With multiple generators, enumeration order is that of nested loops with the rightmost generator innermost:
```miranda
miranda> [ (x, y) | x <- [1,2]; y <- [1,2,3] ]
Result: [(1,1),(1,2),(1,3),(2,1),(2,2),(2,3)]
```

Generator variables scope left to right, so later generators may depend on earlier ones. The classic example — all permutations of a list, using the `--` difference operator:
```miranda
perms [] = [[]]
perms x = [ a : p | a <- x; p <- perms (x -- [a]) ]

miranda> perms [1,2,3]
Result: [[1,2,3],[1,3,2],[2,1,3],[2,3,1],[3,1,2],[3,2,1]]
```

Note that comprehensions do not remove duplicates from their result, and a comprehension over an infinite first generator is itself a perfectly good infinite list.
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

A Miracula script is a text file ending in `.m` containing a list of definitions. Definitions are type-checked top-to-bottom: a definition may refer to itself (recursion), to built-ins, and to identifiers defined *earlier* in the file, but not to ones defined later. Within a single definition, `where`-clause bindings may refer to each other freely.

Top-level definitions are *memoized* (constant applicative forms): a definition without parameters is evaluated at most once per session, on first use, and every later reference returns the cached value. A self-referential constant such as `x = x + 1` is detected and reported as `Infinite loop on identifier: x`.

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
`where` bindings are mutually recursive within one definition, may use patterns and multiple equations themselves, and may nest further `where` clauses under deeper indentation.

## Destructuring bindings

A `where` or `let` binding may have a tuple or cons **pattern** on the left, binding all of its variables at once (the right-hand side is evaluated once and shared):
```miranda
roots a b c = (lo, hi)
              where
              (lo, hi) = (a - b, a + c)

firstTwo xs = a + b where (a:b:rest) = xs
```
This replaces the Miranda idiom of writing a projection helper (`fst`, `snd`) for every tuple result — handy for functions that return a pair, such as a state-threading step.

## Guarded equations

The right-hand side of an equation may be split into *guarded clauses*. Each clause is an expression followed by a comma and either `if condition` or `otherwise`; the clauses are tried top to bottom and the first whose condition holds is chosen:

```miranda
classify n = "negative", if n < 0
           = "zero", if n == 0
           = "positive", otherwise

miranda> classify (-7)
Result:
negative
```

Guards combine with recursion and with a trailing `where` clause (the `where` scopes over *all* clauses):

```miranda
gcdm a b = gcdm (a - b) b, if a > b
         = gcdm a (b - a), if a < b
         = a, otherwise

bmi w h = q, if q > 0
        = 0 - q, otherwise
          where q = w / (h * h)
```

Guards desugar into a chain of conditionals. Two rules to keep in mind:

1. `otherwise` is not an identifier — it is guard syntax meaning "always true". Every guarded equation should normally end with an `otherwise` clause.
2. **Guards do not fall through to the next equation.** If every guard of the selected equation fails, evaluation stops with `Runtime Error: Pattern matching exhausted` — the next equation is *not* tried (this differs from Miranda):

```miranda
h n = 1, if n > 10
h n = 2

miranda> h 5
Runtime Error: Pattern matching exhausted
```

---

# 15. Pattern matching

Functions can be defined across multiple equations using pattern matching. Equations are tried top to bottom; the first whose patterns all match is chosen:
```miranda
take 0 xs     = []
take n []     = []
take n (x:xs) = x : take (n-1) xs
```

The supported pattern forms are:

| Pattern | Matches |
| --- | --- |
| `42`, `'a'` | the literal integer / character |
| `True`, `False` | the boolean constants |
| `x` | anything (binds `x`) |
| `_` | anything (binds nothing) |
| `[]` | the empty list |
| `(x:xs)` | a non-empty list (head and tail; nestable: `(x:y:rest)`) |
| `(p1, p2, …)` | a tuple, element-wise (patterns nest freely inside: `(x:xs, n)`) |

Verified examples:
```miranda
swap (a, b)      = (b, a)
firstTwo (x:y:_) = (x, y)
bnot True        = False
bnot False       = True
secondOf (_, y)  = y

miranda> firstTwo [7,8,9]
Result: (7,8)
```

Patterns combine freely with guards — the pattern selects the equation, the guards select within it:
```miranda
lastx [] = 0 - 1
lastx (a:rest) = a, if rest == []
              = lastx rest, otherwise

miranda> lastx [5,6,7]
Result: 7
```

Limitations:
- Non-empty bracketed list patterns are not supported: write `(x:y:[])` instead of `[x, y]`.
- A repeated variable in one pattern, e.g. `same (x, x) = x`, is accepted but performs **no equality check** — the leftmost occurrence silently wins (`same (1, 2)` is `1`). In Miranda such a pattern matches only when the components are equal.
- Patterns appear only on equation left-hand sides (top-level and in `where` clauses) and in comprehension generators (`(k, v) <- pairs`); a generator whose pattern fails simply skips that element.
- If no equation matches, evaluation stops with `Runtime Error: Pattern matching exhausted`.

Internally, multi-equation pattern definitions are desugared into a decision tree of conditionals and projections over plain lambda calculus.

---

# 16. Basic type structure

Miracula is statically typed: every definition and REPL expression is checked by Hindley–Milner type inference before it is evaluated, so type errors are reported at load time with the offending source position. There are no type declarations — types are inferred.

The type formers are:

| Type | Meaning | Printed as |
| --- | --- | --- |
| `num` | 64-bit signed integer | `Int` |
| `bool` | `True` / `False` | `Bool` |
| `char` | character (strings are `[char]`) | `Char` |
| `[t]` | list of `t` | `[Int]`, `[[Char]]`, … |
| `(t1, t2, …)` | tuple | `(Int, Bool)` |
| `t1 -> t2` | function | `Int -> Int` |
| map | associative map, keys `num` or `[char]` | `Map(a, b)` |
| set | membership set | `Set(a)` |
| vec | vector with O(1) indexed access | `Vec(a)` |
| pq | priority queue (min-heap, integer priorities) | `PQ(a)` |

Type variables print as `a`, `b`, `c`, …. Polymorphic definitions are generalised automatically, e.g. the inferred type of `map` is `(a -> b) -> [a] -> [b]`.

A type error looks like:

```
example.m:3:9: Type Error: cannot unify Int and [Char]
  3 | bad x = x + "one"
    |         ^
```

At run time the evaluator still validates operand shapes (e.g. arithmetic requires integers, comparison recursively compares lists, tuples, integers, and characters), so evaluating a value of the wrong shape raises a runtime error rather than corrupting evaluation.

---

# 17. The special function `show`

The built-in function `show` converts any value into its printable string representation (a `[char]`). Numbers, characters, lists, and tuples format structurally; strings inside a structure print quoted, while a string result at the REPL prints as raw text on the line after `Result:`.

`show` is fully polymorphic — unlike Miranda, there is no monomorphism restriction on its use in scripts — and it is first-class, so it can be passed around like any function:

```miranda
miranda> map show [1,2,3]
Result: ["1","2","3"]
```

Values with no useful textual form print as placeholders: functions as `\x. <closure>`, maps as `<map>`, sets as `<set>` (vectors print their elements: `vec [1,2,3]`). Applying `show` to an infinite list never terminates — take a prefix first.

---

# 18. Miracula lexical syntax

- **Comments**: Comments start with `||` and continue to the end of the line.
- **Identifiers**: Letters, digits, and underscores, starting with a letter or underscore; lowercase initial for variables, uppercase for constructors (`True`, `False`).
- **Keywords**: `if`, `then`, `else`, `ifzero`, `mod`, `where` are reserved. `otherwise` is only special inside guard clauses.
- **Character escapes**: in character literals `'\n'`, `'\t'`, `'\''`, `'\\'`; in string literals `\n`, `\t`, `\"`, `\\`. Any other escaped character stands for itself.
- **Unrecognised symbols** (`$`, `%`, `^`, `@`, a lone `!`, …) outside strings and comments are a parse error, reported with the exact source position and a caret.

---

# 19. The standard environment

The standard library `stdenv.m` is automatically loaded at startup and defines common operations. Its core is shown below; a further set of functions adapted from the Miranda standard environment follows it in the file (listed after the code).

```miranda
|| string == [char]

|| strict in the accumulator (via seq) so long folds run in constant space
foldl f z []     = z
foldl f z (x:xs) = seq z2 (foldl f z2 xs)
                   where
                   z2 = f z x

converse f a b = f b a

sum = foldl (+) 0

product = foldl mul 1
          where mul a b = a * b

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

`stdenv.m` then adds the parts of the Miranda standard environment that are
expressible in Miracula:

- **combinators**: `id`, `const`, `fst`, `snd`
- **numeric** (integer only): `neg`, `abs`, `subtract`
- **logical folds**: `and`, `or`
- **list building**: `concat`, `postfix`
- **non-empty folds**: `foldl1`, `foldr1`
- **ordering** (structural, section 7): `max2`, `min2`, `max`, `min`, `merge`, `sort`, `mkset`
- **character predicates**: `digit`, `letter`
- **more list processing**: `dropwhile`, `index`, `init`, `last`, `limit`, `until`, `scan`, `map2`, `zipWith`, `zip2`, `transpose`
- **text formatting**: `rep`, `spaces`, `ljustify`, `rjustify`, `cjustify`, `lay`, `layn`
- **wider zips**: `zip3`, `zip4`, `zip5`, `zip6`

Parts of the Miranda environment that Miracula cannot express are recorded (with
the reason) in a commented block at the end of `stdenv.m` — chiefly the
floating-point and transcendental functions (no float type), `error`/`undef`,
`code`/`decode`, the UNIX interface, and the `show*` number formatters. Miranda's
list-membership `member` is also omitted because that name is the native
set-membership builtin; define `elem` in your own script if you need it.

---

# 20. Some hints on Miracula style

These are suggestions, not rules — adapted from the Miranda manual's guidelines and the experience of writing the Advent of Code solvers in this repository.

**Prefer comprehensions, ranges, and folds to ad-hoc recursion.** A script made of many small functions recursing into each other is the functional equivalent of spaghetti. Compare:

```miranda
fac n = product [1..n]
```
with the first-principles recursion it replaces. Likewise a Cartesian product as a comprehension,
```miranda
cp x y = [ (a, b) | a <- x; b <- y ]
```
is clearer than the two-level recursion needed to write it by hand. `map`, `filter`, `foldl`, `foldr`, `take`, `drop`, and `takewhile` capture the common recursion patterns; reach for them first, and for the native builtins (section 22) when data sizes grow.

**Keep `where` nesting shallow.** One level of local definitions per top-level function reads well; `where` inside `where` should be rare. Deeply nested helpers cannot be exercised separately, and in the REPL that matters: a helper defined at top level can be tested interactively on its own input the moment it is defined. If a helper does not need the enclosing function's variables, make it top-level.

**Order definitions bottom-up.** Since scripts are checked top-to-bottom (section 13), helpers must precede their users; put the small building blocks first and `main` last, and the file reads as a narrative from pieces to whole.

**Document intended types in comments.** There are no type declarations, and inference will happily give a wrong-but-consistent program a surprising type. For any non-obvious top-level function, a one-line comment such as
```miranda
|| dist :: (num,num,num) -> (num,num,num) -> num
```
costs nothing and pins down intent where the compiler cannot.

**Force long accumulators.** The standard `foldl` is already strict, but hand-written accumulating loops should force their accumulator with `seq` each step — an unforced accumulator builds a chain of suspended additions across millions of iterations.

---

# 21. UNIX/Miracula system interface

The following environment-interaction functions are natively supported:
- **`read filename`**: Reads raw text from `filename` and returns it as a character list (string).
- **`lines str`**: Splits a character list (string) on newline characters (`
`), returning a list of strings.
- **`numval str`**: Converts a character list (string) representing an integer into an `Int`.
- **`/e` Editor Config**: Uses the `./mica` editor if present in the workspace, or defaults to the Unix `vi` command.

---

# 22. High-performance built-in functions

To make large-input problems (such as Advent of Code puzzles) practical, Miracula includes general-purpose built-in functions implemented natively in Go. Signatures below use Miranda-style type variables `*`, `**`; the type checker prints the same types with variables `a`, `b` and the constructors `Map(a, b)`, `Set(a)`, `Vec(a)`.

All examples in this section are verified against the current interpreter.

## 22.1 Strict evaluation: `seq`

| Function | Signature | Description |
| --- | --- | --- |
| `seq` | `* -> ** -> **` | Forces its first argument to weak head normal form, then returns the second. |

`seq` pins down evaluation order in an otherwise lazy language — its main use is keeping accumulators strict so long loops run in constant space (the standard `foldl` is built on it, which is why `sum [1..1000000]` works):

```miranda
miranda> seq (1 + 2) "done"
Result:
done
miranda> sum [1..1000000]
Result: 500000500000
```

## 22.2 String processing: `split`, `parse_ints`

| Function | Signature | Description |
| --- | --- | --- |
| `split` | `[char] -> [char] -> [[char]]` | Splits the second string at any character of the first (the delimiter set); empty fields are dropped. |
| `parse_ints` | `[char] -> [num]` | Extracts every (optionally negative) integer from a string. |

```miranda
miranda> split "," "a,bb,ccc"
Result: ["a","bb","ccc"]
miranda> split " ,;" "1, 2;3"
Result: ["1","2","3"]
miranda> parse_ints "x=3, y=-7, z=12"
Result: [3,-7,12]
```

`parse_ints` is usually all that is needed to read a puzzle input:

```miranda
total = sum (parse_ints (read "input.txt"))
```

## 22.3 Maps: `empty_map`, `h_insert`, `h_lookup`, `h_lookup_def`

| Function | Signature | Description |
| --- | --- | --- |
| `empty_map` | `map * **` | The empty map. |
| `h_insert` | `map * ** -> * -> ** -> map * **` | Returns a new map with the key bound to the value; the original map is unchanged. |
| `h_lookup` | `map * ** -> * -> **` | Returns the value for a key; runtime error if absent. |
| `h_lookup_def` | `map * ** -> * -> ** -> **` | Returns the value for a key, or the given default if absent (the default is only evaluated on a miss). |

Maps are immutable AVL trees with structural sharing: `h_insert` is O(log n) and older versions of the map remain valid. Keys are integers or strings (one kind per map — the type checker enforces this). Integer keys are handled natively, without string conversion.

```miranda
miranda> h_lookup (h_insert (h_insert empty_map "ada" 36) "alan" 41) "alan"
Result: 41
miranda> h_lookup_def empty_map "grace" 0
Result: 0
```

Building a map by folding — 50,000 inserts complete in well under a second:

```miranda
squares = foldl ins empty_map [1..10]
          where ins m k = h_insert m k (k * k)

miranda> h_lookup squares 7
Result: 49
```

## 22.4 Sets: `empty_set`, `s_insert`, `member`

| Function | Signature | Description |
| --- | --- | --- |
| `empty_set` | `set *` | The empty set. |
| `s_insert` | `set * -> * -> set *` | Returns a new set with the element added; the original set is unchanged. |
| `member` | `set * -> * -> bool` | Membership test. |

Sets share the persistent AVL representation of maps: `s_insert` is O(log n) with structural sharing, and elements are integers or strings (one kind per set). The classic visited-set idiom:

```miranda
visited = foldl s_insert empty_set [3, 7, 3, 12]

miranda> (member visited 7, member visited 4)
Result: (True,False)
```

## 22.5 Vectors: `to_vec`, `vec_get`, `vec_set`, `vec_len`, `vec_to_list`

| Function | Signature | Description |
| --- | --- | --- |
| `to_vec` | `[*] -> vec *` | Materialises a list as a vector (elements stay lazy). |
| `vec_get` | `vec * -> num -> *` | O(1) indexed read (0-based; bounds checked). |
| `vec_set` | `vec * -> num -> * -> vec *` | Returns a new vector with one element replaced; the original is unchanged. |
| `vec_len` | `vec * -> num` | O(1) length. |
| `vec_to_list` | `vec * -> [*]` | Converts back to a list. |

Vectors give constant-time random access where list indexing is linear — the right structure for grids and tables:

```miranda
miranda> vec_get (to_vec [10,20,30]) 1
Result: 20
miranda> vec_len (to_vec [10,20,30])
Result: 3
miranda> vec_to_list (vec_set (to_vec [10,20,30]) 0 99)
Result: [99,20,30]
```

Because vectors are persistent, updating one never disturbs earlier references:

```miranda
v = to_vec [10,20,30]
w = vec_set v 0 99

miranda> (vec_get v 0, vec_get w 0)
Result: (10,99)
```

The older `list_get :: [num] -> num -> num` and `list_set :: [num] -> num -> num -> [num]` builtins remain for compatibility, but they convert the whole list on every call (O(n)); prefer vectors.

## 22.6 Sorting: `sort_ints`, `sort_by`, `sort_edges`, `sort_pts`

| Function | Signature | Description |
| --- | --- | --- |
| `sort_ints` | `[num] -> [num]` | Ascending sort of integers. |
| `sort_by` | `(* -> * -> num) -> [*] -> [*]` | Sort with a comparison function returning negative / zero / positive. |
| `sort_edges` | `[(num,num,num)] -> [(num,num,num)]` | Sorts triples ascending by their third component (e.g. weighted edges by distance). |
| `sort_pts` | `[(num,(num,num,num))] -> [(num,(num,num,num))]` | Sorts indexed 3-D points ascending by their x coordinate. |

`sort_edges` and `sort_pts` extract their integer keys once and sort natively, so sorting hundreds of thousands of tuples is fast.

```miranda
miranda> sort_ints [3,1,2]
Result: [1,2,3]
miranda> sort_by (\a. \b. b - a) [3,1,2]
Result: [3,2,1]
miranda> sort_edges [(1,2,9),(3,4,1)]
Result: [(3,4,1),(1,2,9)]
```

Because the ordering operators are polymorphic (section 7), `sort_by` sorts any
comparable element — strings, tuples, and so on — with a three-way comparator
built from `<`:

```miranda
before a b = 0 - 1, if a < b
           = 1, if b < a
           = 0, otherwise

miranda> sort_by before ["pear", "fig", "apple", "date"]
Result: ["apple","date","fig","pear"]
miranda> sort_by before [(2,1), (1,9), (1,2), (2,0)]
Result: [(1,2),(1,9),(2,0),(2,1)]
```

## 22.7 Memoization: `memoize`

| Function | Signature | Description |
| --- | --- | --- |
| `memoize` | `(* -> **) -> (* -> **)` | Wraps a function so results are cached by argument; integer arguments are cached without serialization. |

```miranda
steps n = if n == 1 then 0
          else (if n mod 2 == 0 then 1 + steps (n / 2)
                else 1 + steps (3 * n + 1))
msteps = memoize steps

miranda> msteps 27 + msteps 27
Result: 222
```

The first call computes; the second returns the cached result. Note that scripts are type-checked top-to-bottom, so a recursive function cannot refer to its own memoized wrapper defined later — `memoize` caches whole top-level calls.

`memofix` memoizes a *recursive* function without threading a cache by hand. It takes an **open** recursion — a function whose first parameter is the recursive call — and returns the memoized fixpoint, so the recursive calls hit the cache too:

| Function | Signature | Description |
| --- | --- | --- |
| `memofix` | `((* -> **) -> * -> **) -> (* -> **)` | Memoized open recursion: `memofix f` is the least function `g` with `g x = f g x`, caching by `x`. |

```miranda
fib = memofix f
      where f rec n = n, if n < 2
                    = rec (n - 1) + rec (n - 2), otherwise

miranda> fib 40
Result: 102334155
```

For a DP over several inputs, make the argument a tuple: `count = memofix f where f rec (a, b) = … rec (a', b') …`. This is how the Plutonian-pebbles solver ([aoc24/aoc24-11.m](aoc24/aoc24-11.m)) counts stones after 75 blinks with no memo map in sight.

## 22.8 Bitwise operators: `xor`, `band`, `bor`, `shl`, `shr`

There are no bitwise infix operators; these builtins act on the 64-bit integer representation.

| Function | Signature | Description |
| --- | --- | --- |
| `xor` | `num -> num -> num` | bitwise exclusive-or |
| `band` | `num -> num -> num` | bitwise and (`and`/`or` are the stdenv list folds) |
| `bor` | `num -> num -> num` | bitwise or |
| `shl` | `num -> num -> num` | left shift (`shl a n` = `a` × 2ⁿ) |
| `shr` | `num -> num -> num` | right shift (arithmetic) |

```miranda
miranda> xor 12 10
Result: 6
miranda> (shl 1 4, shr 255 4, bor (shl 1 3) (shl 1 5))
Result: (16,15,40)
```

A common idiom is a bitmask visited-set: `seen2 = bor seen (shl 1 i)`, tested with `band seen (shl 1 i) ~= 0`.

## 22.9 Priority queues: `pq_empty`, `pq_push`, `pq_pop`, `pq_null`

A persistent min-heap keyed by an integer priority — the right structure for Dijkstra / A* frontiers.

| Function | Signature | Description |
| --- | --- | --- |
| `pq_empty` | `pq *` | the empty queue |
| `pq_push` | `pq * -> num -> * -> pq *` | queue with `(priority, value)` added; the original is unchanged |
| `pq_pop` | `pq * -> (num, *, pq *)` | the lowest-priority `(priority, value)` and the rest of the queue; runtime error if empty |
| `pq_null` | `pq * -> bool` | is the queue empty |

```miranda
drain pq = if pq_null pq then []
           else let (p, v, rest) = pq_pop pq in (p, v) : drain rest

miranda> drain (pq_push (pq_push (pq_push pq_empty 3 'c') 1 'a') 2 'b')
Result: [(1,'a'),(2,'b'),(3,'c')]
```

The Reindeer-maze solver ([aoc24/aoc24-16.m](aoc24/aoc24-16.m)) runs Dijkstra over `(cell, direction)` states with `pq_pop`/`pq_push` and reads the popped triple with a destructuring `let`.

## 22.10 A complete worked example

The Advent of Code 2025 Day 8 solver ([aoc8.m](aoc8.m)) combines most of these: `read` + `parse_ints` for input, a list comprehension over all point pairs, `sort_edges` + `take` for the 1000 shortest, and maps for union-find:

```miranda
pts      = group3 (parse_ints input)
edges    = [ (i, j, distSq p q) | (i, p) <- ipts; (j, q) <- ipts; i < j ]
shortest = take 1000 (sort_edges edges)
```

Its 499,500 pairwise distances evaluate, sort, and cluster in about 1.3 seconds.

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

# 24. Miracula for Miranda users

If you already know Miranda, Miracula will feel immediately familiar: lazy evaluation, `||` comments, the offside layout rule, guarded equations with `otherwise`, `where` clauses, list comprehensions with `;`-separated qualifiers, sections, `#`, `++`, `--`, `:`, and strings as `[char]` all work as you expect. This section lists only what is *different*. Every claim below is verified against the current interpreter.

## Things Miranda has that Miracula does not

| Miranda | In Miracula |
| --- | --- |
| floats; arbitrary-precision integers | `num` is a 64-bit signed integer only; overflow wraps silently |
| `x div y` and float `/` | `/` *is* integer division (floor); there is no `div` |
| `x ^ y`, `xs ! n` | not supported — use `vec_get (to_vec xs) n` for subscripting |
| type declarations `f :: num -> num` | parse error — types are inferred only (Hindley–Milner, section 16) |
| algebraic types `tree ::= Leaf \| Node ...` | not supported; model variants with tuples/tags |
| type synonyms `string == [char]`, `abstype` | not supported |
| `%include`, `%export`, literate scripts | no module system; one script + `stdenv.m` |
| order-independent definitions | **checked top-to-bottom**: a definition may not reference one defined later in the file, so mutual recursion across top-level definitions is rejected |
| guard fall-through to the next equation | **failing all guards is a runtime error** (`Pattern matching exhausted`) — always end with `otherwise` |
| continued relations `0 <= x < 10` | syntax error — write `0 <= x & x < 10` |
| n+k patterns `f (n+1) = ...` | parse error — match on `n` and use `n - 1` |
| repeated pattern variables `(x, x)` match only equal parts | accepted but **no equality check** — the leftmost binding silently wins |
| multi-variable generators `a,b <- xs` | parse error — write `a <- xs; b <- xs` |
| recurrence generators `x <- a, f x ..` | parse error — use `iterate f a` |
| `show` restricted to monomorphic contexts in scripts | no restriction — `show` is fully polymorphic everywhere |
| list patterns `[a, b]` | parse error — write `(a:b:[])` |
| tuple bindings in where: `(a, b) = e` | parse error — use a pattern-matching helper: `first (a, b) = a` |
| left sections `(1+)`, `(2/)` | only *right* sections `(op e)` and bare `(op)` (section 8) — for a left section write a lambda |
| `$fn` user-defined infix | not supported |
| `error`, `undef` | not available |
| curried `zip2 xs ys` | `zip` takes a *tuple* of lists: `zip (xs, ys)` |
| rich stdenv (`abs`, `max`, `concat`, `last`, …) | minimal stdenv (section 19) — define what you need or use the native builtins |

## Things Miracula has that Miranda does not

| Feature | Example |
| --- | --- |
| lambda abstractions | `map (\x. x * x) [1..3]` (Miranda has no anonymous functions) |
| conditional expressions | `if p then a else b` and `ifzero n then a else b` (Miranda uses guards only) |
| the pipe operator | `"peter" \|> reverse \|> hd` → `'r'` (section 6) |
| `!=` | accepted as an alias for `~=` |
| native maps, sets, vectors | `h_insert`/`h_lookup`, `s_insert`/`member`, `to_vec`/`vec_get` — persistent, O(log n)/O(1) (section 22) |
| native input parsing | `read`, `lines`, `split`, `parse_ints` — `sum (parse_ints (read "input.txt"))` |
| native sorting and memoization | `sort_ints`, `sort_by`, `sort_edges`, `sort_pts`, `memoize` |
| REPL conveniences | tab completion, `?identifier` source lookup, per-expression timing, Ctrl-C interruption of a running evaluation |

Two behavioural notes that have no syntax at all: top-level definitions are memoized once per session (section 13), and evaluation depth is bounded only by memory — a `foldr` over millions of elements evaluates rather than overflowing a stack.

---

# 25. Miracula for Haskell users

Coming from Haskell, the semantics will feel like home: call-by-need laziness (with sharing), currying and partial application, Hindley–Milner type inference, `where` clauses, guards with `otherwise`, list comprehensions, `[a..b]` and `[a..]` ranges, cons `:`, append `++`, composition `.`, `seq`, and `String = [Char]`. The syntax, however, is Miranda's — and three Haskell habits are outright traps.

## Three silent or confusing traps

1. **`||` starts a comment.** Haskell's logical OR is Miracula's end-of-line comment, so `a || b` silently evaluates to just `a` — no error. Write `a \/ b`:
```miranda
miranda> True || False      || everything after the bars is a comment
Result: True
miranda> True \/ False
Result: True
```
2. **`--` is list difference**, not a comment (it is Haskell's `Data.List.\\`): `[1,2,3] -- [2]` is `[1,3]`. Comments are `||`.
3. **`/=` is not an operator.** It lexes as `/` followed by `=`, and since `=` marks a definition you get a baffling "left hand side of binding" parse error. Write `~=` (or `!=`).

## Translation table

| Haskell | Miracula |
| --- | --- |
| `\x y -> e` | `\x. \y. e` (dot, one variable per lambda) |
| `a && b`, `a \|\| b`, `not a` | `a & b`, `a \/ b`, `~a` |
| `a /= b` | `a ~= b` or `a != b` |
| `-- comment` | `\|\| comment` |
| `Data.List.\\` | `--` (infix list difference) |
| `head`, `tail`, `fst`, `snd` | `hd`, `tl`; define `fst (a, b) = a` yourself |
| `f x \| x > 0 = e` | `f x = e, if x > 0` (guards come after `=`, Miranda style) |
| `let x = e in b` | not supported — use a `where` clause |
| `case e of ...` | not supported — use multi-equation definitions with patterns/guards |
| `x `div` y` (backticks) | `x / y` (`/` *is* floor integer division; backticks are a lex error) |
| `xs !! n` | `vec_get (to_vec xs) n` |
| `[x \| x <- xs, p x]` | `[x \| x <- xs; p x]` (semicolons between qualifiers) |
| `zip xs ys` | `zip (xs, ys)` (one tuple argument), or the curried `zip2 xs ys` / `zipWith f xs ys` |
| `f $ g x` | parens, or flip the flow: `x \|> g \|> f` (`$` is a lex error) |
| `Data.Function.&` | `\|>` (and note `&` here means AND) |
| `f :: a -> b` | not supported — inference only, no annotations |
| `data` / `newtype` / `type` / classes | not supported — tuples and tags |
| `import` / modules / `do` / `IO` | none: one script; `main` is a value that gets printed; `read "file"` returns the file contents as a string |
| `Integer` (bignum), `Double` | only `num` = 64-bit signed integer; overflow wraps |
| `[1,3..9]` | `[1, 3 .. 9]` = `[1,3,5,7,9]` (stepped range; needs the comma) |
| `[a, b]` as a *pattern* | `(a:b:[])` |
| `x@(y:ys)`, `~pat`, records | not supported (but `\(a,b). e` pattern lambdas work) |
| left sections `(2*)`, `(subtract 2)` | right sections `(op e)` and bare `(op)` are supported; write a lambda for a left section |
| `chr` / `ord` | same names: `chr :: num -> char`, `ord :: char -> num` |
| `Data.Map` / `Data.Set` / arrays | native `h_insert`/`h_lookup`, `s_insert`/`member`, `to_vec`/`vec_get` (section 22) |

## Semantics worth knowing

- Definitions are type-checked **top-to-bottom**: no forward references and no mutual recursion between top-level definitions (unlike Haskell's whole-module scope). Within one definition, `where` bindings are mutually recursive as usual.
- If every guard of an equation fails, evaluation stops with `Pattern matching exhausted` — guards do not fall through to the next equation, so end guarded equations with `otherwise`.
- Top-level definitions are CAFs evaluated at most once per session, and evaluation depth is bounded only by memory, so deep `foldr`s over millions of elements are fine.
- `foldl` in the standard environment is strict in its accumulator (Haskell's `foldl'`).

---

# 26. Miracula for Admiran users

Admiran and Miracula are sibling Miranda descendants, so a lot transfers directly: `||` line comments, guards written `expression, if condition` with `otherwise`, `where` clauses, `\/` / `&` / `~` logic, the pipe operator `|>`, lazy evaluation with `seq`, strings as `[char]`, list comprehensions, `[a..b]` and infinite `[a..]` ranges — and both replaced Miranda's arbitrary-precision `num` with 64-bit signed integers. The big split: Admiran is a self-hosting *compiler* producing native executables from module trees; Miracula is an interpreter with an interactive REPL and a single-script model.

## Things Admiran has that Miracula does not

| Admiran | In Miracula |
| --- | --- |
| `%import` / `%export`, qualified modules | no module system — one script plus `stdenv.m` |
| type declarations `f :: type` | parse error — types are inferred only |
| algebraic types `t * ::= C1 \| C2 ...`, strict fields, `abstype`, `==` synonyms | not supported — tuples and tags |
| `case e of ...` | not supported — multi-equation definitions with patterns/guards |
| block comments `{\| ... \|}` | lex error — only `\|\|` line comments |
| `\x y -> e` (arrow, multi-var) | `\x. \y. e` (dot, one parameter per lambda; the parameter may be a tuple/cons pattern) |
| chainable comparisons `a < b < c` | syntax error — write `a < b & b < c` |
| `x $div y`, `$fn` infix syntax | lex error — `/` *is* floor integer division, `mod` is an infix keyword |
| `^` power | parse error — none built in; bitwise ops are the builtins `xor`/`band`/`bor`/`shl`/`shr` (section 22) |
| `xs ! n`, `xs !! n` indexing | `vec_get (to_vec xs) n` |
| hex/octal/binary literals `0xff` | **pitfall**: `0xff` lexes as `0` applied to a variable `xff` ("unbound variable: xff") — decimal only |
| unboxed `42#` values | no unboxed values (`#` is prefix length) |
| `$` / `$!` application operators | parens or `\|>`; force with `seq` |
| `.>` reverse composition | chain with `\|>` instead |
| per-type `show*`/`cmp*` instances and dictionaries | not needed: `show`, `==`, and the comparison operators are polymorphic and structural, as in Miranda |

## Things Miracula has that Admiran does not

- An interactive REPL: tab completion, `?identifier` source lookup, per-expression timing, Ctrl-C interruption, and definitions persisted to `~/.script.m` (sections 4–5).
- Polymorphic structural equality and `show` over any value — no instance plumbing.
- Native data structures and helpers aimed at puzzle-scale input crunching: persistent maps and sets, O(1) vectors, `split`/`parse_ints`, native sorts, and `memoize` (section 22).
- `--` as infix list difference, and `!=` as an alias for `~=`.

## Semantics worth knowing

- Definitions are type-checked **top-to-bottom** within the one script: no forward references and no mutual recursion between top-level definitions.
- If every guard of an equation fails, evaluation stops with `Pattern matching exhausted` rather than falling through to the next equation.
- Top-level definitions are CAFs (evaluated at most once per session), `foldl` in the standard environment is strict in its accumulator, and evaluation depth is bounded only by memory.

---

# 27. License

Copyright Research Software Limited 1985-2020. Adapted for Miracula (a Go subset implementation of Miranda).

---

# 28. Bug reports

Please report any interpreter bugs, parser errors, or REPL issues to the project maintainer.
