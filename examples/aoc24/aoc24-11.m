|| Advent of Code 2024 - Day 11: Plutonian Pebbles
|| https://adventofcode.com/2024/day/11
|| Each blink transforms every stone: a 0 becomes 1; a stone with an even
|| number of digits splits into its two halves; otherwise it is multiplied by
|| 2024. Part 1 counts the stones after 25 blinks, part 2 after 75. The count
|| explodes, so we memoize count(stone, steps) in a map threaded through the
|| recursion (a string key "stone|steps"), like aoc25/aoc11.m threads its map.
||
|| inputs/day11.txt is seeded with the official example (part 1 answer 55312);
|| run fetch-inputs.sh to replace it with your personal puzzle input.

ndigits n = if n < 10 then 1 else 1 + ndigits (n / 10)
pow10 0 = 1
pow10 k = 10 * pow10 (k - 1)

|| count stones produced by one stone after `steps` blinks, threading memo map.
|| a single flat `where` (nested where referencing outer bindings is buggy in
|| this interpreter); every branch binding is lazy, so only its branch runs.
count stone steps memo
    = (1, memo), if steps == 0
    = (cached, memo), if cached ~= 0 - 1
    = (total, h_insert m2 key total), otherwise
      where
      key = show stone ++ "|" ++ show steps
      cached = h_lookup_def memo key (0 - 1)
      nd = ndigits stone
      half = pow10 (nd / 2)
      res = zero, if stone == 0
          = (fst lr + fst rr, snd rr), if nd mod 2 == 0
          = count (stone * 2024) (steps - 1) memo, otherwise
      zero = count 1 (steps - 1) memo
      lr = count (stone / half) (steps - 1) memo
      rr = count (stone mod half) (steps - 1) (snd lr)
      total = fst res
      m2 = snd res

|| sum counts over all starting stones, threading the memo across them
sumStones [] steps memo acc = acc
sumStones (s:ss) steps memo acc = sumStones ss steps m2 (acc + c)
                                  where
                                  r = count s steps memo
                                  c = fst r
                                  m2 = snd r

stones = parse_ints (read "examples/aoc24/inputs/day11.txt")

solvePart1 = sumStones stones 25 empty_map 0
solvePart2 = sumStones stones 75 empty_map 0

main = "Advent of Code 2024 - Day 11 Results:\n" ++
       "  Part 1 (stones after 25 blinks): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (stones after 75 blinks): " ++ show solvePart2 ++ "\n"
