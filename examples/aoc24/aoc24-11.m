|| Advent of Code 2024 - Day 11: Plutonian Pebbles
|| https://adventofcode.com/2024/day/11
|| Each blink transforms every stone: a 0 becomes 1; a stone with an even
|| number of digits splits into its two halves; otherwise it is multiplied by
|| 2024. Part 1 counts the stones after 25 blinks, part 2 after 75. The count
|| explodes, so count(stone, steps) is memoised with `memofix` — open recursion
|| whose `rec` calls hit a hidden cache, so no memo map is threaded by hand.
||
|| inputs/day11.txt is seeded with the official example (part 1 answer 55312);
|| run fetch-inputs.sh to replace it with your personal puzzle input.

ndigits n = if n < 10 then 1 else 1 + ndigits (n / 10)
pow10 0 = 1
pow10 k = 10 * pow10 (k - 1)

|| stones produced by one stone after `steps` blinks (memoised open recursion)
countStep rec (stone, steps)
    = 1, if steps == 0
    = rec (1, steps - 1), if stone == 0
    = rec (stone / half, steps - 1) + rec (stone mod half, steps - 1), if nd mod 2 == 0
    = rec (stone * 2024, steps - 1), otherwise
      where
      nd = ndigits stone
      half = pow10 (nd / 2)
count = memofix countStep

stones = parse_ints (read "examples/aoc24/inputs/day11.txt")

solvePart1 = sum [count (s, 25) | s <- stones]
solvePart2 = sum [count (s, 75) | s <- stones]

main = "Advent of Code 2024 - Day 11 Results:\n" ++
       "  Part 1 (stones after 25 blinks): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (stones after 75 blinks): " ++ show solvePart2 ++ "\n"
