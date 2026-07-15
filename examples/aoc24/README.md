# Advent of Code 2024 in Miracula

Solutions to all 25 days of [Advent of Code 2024](https://adventofcode.com/2024)
written in Miracula (the Miranda-inspired language in this repo — see
[`miracula-man.md`](../../miracula-man.md)). Each day is one script,
`aoc24-N.m`, printing both parts.

The solutions lean on the language's Advent-of-Code extensions (section 22 of
the manual): `read` / `lines` / `split` / `parse_ints` for input, native
persistent maps (`h_insert`/`h_lookup`), sets (`s_insert`/`member`), vectors
(`to_vec`/`vec_get`), sorting (`sort_ints`/`sort_by`), and list comprehensions,
ranges, and folds throughout.

## Running

Run from the **repository root** (input paths and `stdenv.m` resolve against the
current directory):

```
./mira -xt examples/aoc24/aoc24-1.m      # result + timing
./mira -x  examples/aoc24/aoc24-1.m      # result only
```

Build `mira` first if needed: `make build`.

## Inputs

`inputs/dayN-example.txt` holds the official worked example for each day, and
`inputs/dayN.txt` is **seeded with that same example** so every solver runs out
of the box and reproduces the puzzle's published answer.

Advent of Code asks that personal puzzle inputs not be redistributed, so your
own inputs are not included. To solve with your inputs, fetch them with your
session cookie:

```
AOC_SESSION=<your adventofcode.com session cookie> sh examples/aoc24/fetch-inputs.sh
```

This overwrites `inputs/dayN.txt` with your real input (the `-example` files are
left alone). A few days also need a grid-size or threshold constant switched
from the example's value to the real one — this is flagged in the table and at
the top of the affected scripts.

## Results (on the seeded example inputs)

All timings are for the small **example** inputs on the reference machine and
are well under a second. Real-input timings are not shown because the personal
inputs are not available here (see above); the algorithms were chosen to scale
(memoisation, Dijkstra, BFS, near-linear scans), but their full-input runtime
has not been measured. The three heaviest days are called out under Notes.

| Day | Puzzle | Part 1 | Part 2 | Example time |
| --- | --- | --- | --- | --- |
| [1](aoc24-1.m) | Historian Hysteria | 11 | 31 | 0 ms |
| [2](aoc24-2.m) | Red-Nosed Reports | 2 | 4 | 0 ms |
| [3](aoc24-3.m) | Mull It Over | 161 | 48¹ | 0 ms |
| [4](aoc24-4.m) | Ceres Search | 18 | 9 | 1 ms |
| [5](aoc24-5.m) | Print Queue | 143 | 123 | 0 ms |
| [6](aoc24-6.m) | Guard Gallivant | 41 | 6 | 4 ms |
| [7](aoc24-7.m) | Bridge Repair | 3749 | 11387 | 0 ms |
| [8](aoc24-8.m) | Resonant Collinearity | 14 | 34 | 0 ms |
| [9](aoc24-9.m) | Disk Fragmenter | 1928 | 2858 | 0 ms |
| [10](aoc24-10.m) | Hoof It | 36 | 81 | 12 ms |
| [11](aoc24-11.m) | Plutonian Pebbles | 55312 | 65601038650482 | 34 ms |
| [12](aoc24-12.m) | Garden Groups | 1930 | 1206 | 2 ms |
| [13](aoc24-13.m) | Claw Contraption | 480 | 875318608908 | 0 ms |
| [14](aoc24-14.m) | Restroom Redoubt | 12 | —² | 2 ms |
| [15](aoc24-15.m) | Warehouse Woes | 10092 | 9021 | 11 ms |
| [16](aoc24-16.m) | Reindeer Maze | 7036 | 45 | 28 ms |
| [17](aoc24-17.m) | Chronospatial Computer | 4,6,3,5,6,3,5,2,1,0 | 117440³ | 0 ms |
| [18](aoc24-18.m) | RAM Run | 22 | 6,1 | 1 ms |
| [19](aoc24-19.m) | Linen Layout | 6 | 16 | 0 ms |
| [20](aoc24-20.m) | Race Condition | 1 / 285⁴ | | 308 ms |
| [21](aoc24-21.m) | Keypad Conundrum | 126384 | 154115708116294 | 19 ms |
| [22](aoc24-22.m) | Monkey Market | 37327623 | 23⁵ | 740 ms |
| [23](aoc24-23.m) | LAN Party | 7 | co,de,ka,ta | 1 ms |
| [24](aoc24-24.m) | Crossed Wires | 2024 | —⁶ | 1 ms |
| [25](aoc24-25.m) | Code Chronicle | 3 | (none) | 0 ms |

All Part 1 / Part 2 values above match the answers published in each puzzle's
example, except where a footnote explains otherwise.

## Notes and caveats

1. **Day 3.** The seeded `day3.txt` is the Part 1 example (answer 161); its Part
   2 value happens to also be 161. The proper Part 2 example is
   `inputs/day3-example2.txt` (answer **48**), verified by pointing the script
   at that file.
2. **Day 14.** The example uses an 11×7 grid, so only Part 1 (safety factor 12)
   is meaningful; the Christmas-tree of Part 2 only appears on the real 101×103
   grid. Set `width = 101` and `height = 103` in the script for your input.
3. **Day 17.** The seeded `day17.txt` is the Part 1 example (whose program is not
   self-reproducing, so Part 2 correctly reports `-1`). The Part 2 example is
   `inputs/day17-example2.txt` (answer **117440**), verified separately.
4. **Day 20.** The example's best cheat saves only 76 ps, so with the real
   threshold of 100 the answers are 0 / 0. The script therefore also prints the
   example's own verifiable counts — **1** two-ps cheat saving ≥ 64 and **285**
   twenty-ps cheats saving ≥ 50 — which match the puzzle text.
5. **Day 22.** The seeded `day22.txt` is the Part 1 example (answer 37327623).
   The Part 2 example is `inputs/day22-example2.txt` (answer **23**), verified
   separately; on the Part 1 seeds Part 2 is 24.
6. **Day 24 Part 2 is not solved.** It asks for the four pairs of swapped gates
   that repair a 45-bit ripple-carry adder — a structural analysis specific to
   the real input. The example circuit is not an adder, so there is nothing to
   verify against, and the script leaves Part 2 unimplemented rather than ship an
   unverifiable guess. Part 1 (2024) is solved.

### Heaviest days

Every solver finishes in well under a second on the example inputs. On a full
personal input the three to watch, none expected to approach the two-minute
budget but untested here, are:

- **Day 6 Part 2** — re-simulates the guard's walk with an extra obstacle on each
  of the ~5000 path cells; each walk is loop-detected with a `(cell, direction)`
  set.
- **Day 20** — scans a Manhattan diamond of up to ~800 offsets around every one
  of the ~10000 path cells (already ~0.3 s on the tiny example, the slowest here).
- **Day 22 Part 2** — evolves 2000 secret numbers for every buyer and aggregates
  four-change windows; the bitwise `xor` is implemented on the bits, which is the
  main cost.

## Implementation notes

A few Miracula characteristics shaped these solutions:

- Top-level definitions are checked **top-to-bottom**, so helpers precede their
  users and there is no mutual recursion between top-level definitions (Day 21's
  cost model is built as an iterated table, and Day 23's clique search is a
  single self-recursive enumeration, for this reason).
- Hand-written recursion that threads an integer accumulator is avoided;
  solutions build lists and `sum`/`foldl` them, or thread native maps/sets, which
  is both idiomatic and reliable.
- There are no bitwise operators, so Days 17 and 22 implement `xor` on the bits.
- Comparisons are integer-only, so Day 23 encodes two-letter host names as
  integers to sort and key them.
