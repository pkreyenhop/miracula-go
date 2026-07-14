# Miracula Examples

Classic example scripts from the Miranda distribution (`miranda/ex`),
ported to run with Miracula. Run them **from the repository root** (so
`stdenv.m` is found):

    ./mira -x examples/primes.m

or load one into the REPL and experiment:

    ./mira examples/primes.m
    miranda> take 100 primes

Every file keeps the original's comments and structure, with a
`|| Ported to Miracula:` note describing what changed and a `main` that
prints a demonstration.

## The examples

| File | What it does |
| --- | --- |
| `ack.m` | the Ackermann function (`ack 3 3` = 61) |
| `divmodtest.m` | checks sign properties of `/` and `mod` |
| `edigits.m` | the decimal digits of `e` from an infinite series |
| `fib.m` | naive Fibonacci |
| `fibs.m` | Fibonacci in linear time by indexing into its own result list |
| `hamming.m` | Hamming numbers by merging communicating streams |
| `hanoi.m` | Towers of Hanoi move list |
| `just.m` | paragraph justification with balanced line fill (Bird & Wadler) |
| `matrix.m` | integer matrix package: determinant, adjoint, product |
| `powers.m` | formatted table of powers |
| `primes.m` | the sieve of Eratosthenes as an infinite list |
| `pyths.m` | Pythagorean triples |
| `queens.m` | all 92 solutions to the eight queens problem |
| `queens1.m` | one solution to eight queens by explicit backtracking |
| `quicksort.m` | two-line quicksort |
| `rational.m` | exact rational arithmetic on (num,num) pairs |
| `selflines.m` | a self-describing infinite scroll of lines |
| `set.m` | finite sets as ordered lists |
| `stack.m` | stacks with a custom show function |
| `topsort.m` | topological sort of a partial order |

## Typical adaptations

The ports mostly follow section 24 of the manual (*Miracula for
Miranda users*): `=` becomes `==` in expressions, n+k patterns become
guards, `xs!i` becomes an `nth` helper, `div` becomes `/`, step ranges
become comprehensions, `$infix` becomes prefix application, missing
stdenv functions (`concat`, `and`, `abs`, `index`, `until`,
`transpose`, `layn`, justification helpers, ...) are defined locally,
mutually recursive top-level pairs are merged into one `where` block,
and helpers are ordered before their users (definitions are checked
top-to-bottom).

## Not ported

- `bignum.m`, `golden.m` — arbitrary-precision arithmetic library (and its client)
- `barry.m`, `keith.m` — need floating point, which Miracula does not have
- `treesort.m`, `refoliate.m`, `unify.m`, `polish.m` — need algebraic data types
- `genmat.m` — needs parameterised modules (`%free`)
- `graphics.m`, `parafs.m` — module pair using `%include` and algebraic types
- `box.m`, `makebug.m`, `mrev` — UNIX filters / file writers using Miranda's system interface
- `kate.lit.m` — literate script that is also a LaTeX document
