|| Advent of Code 2024 - Day 13: Claw Contraptions
|| https://adventofcode.com/2024/day/13
|| Each machine has button A (cost 3) and B (cost 1) moving the claw by fixed
|| offsets, and a prize location. Find non-negative press counts a, b solving
||   a*ax + b*bx = px ,  a*ay + b*by = py
|| via Cramer's rule (the buttons are never parallel in the inputs). Part 1
|| caps presses at 100; part 2 adds 10000000000000 to every prize coordinate.
||
|| inputs/day13.txt is seeded with the official example (part 1 answer 480);
|| run fetch-inputs.sh to replace it with your personal puzzle input.

|| all six numbers per machine come out of parse_ints in order:
|| ax ay bx by px py
chunk6 [] = []
chunk6 (a:b:c:d:e:f:rest) = (a, b, c, d, e, f) : chunk6 rest

machines = chunk6 (parse_ints (read "examples/aoc24/inputs/day13.txt"))

|| cost to win one machine, or 0 if unwinnable. `off` shifts the prize,
|| `cap` bounds each press count (0 - 1 means no bound).
cost off cap (ax, ay, bx, by, px0, py0)
    = 3 * a + b, if ok
    = 0, otherwise
      where
      px = px0 + off
      py = py0 + off
      det = ax * by - ay * bx
      na = px * by - py * bx
      nb = ax * py - ay * px
      a = na / det
      b = nb / det
      capok = (cap < 0) \/ (a <= cap & b <= cap)
      ok = det ~= 0 & na mod det == 0 & nb mod det == 0
           & a >= 0 & b >= 0 & capok

solvePart1 = sum [cost 0 100 m | m <- machines]
solvePart2 = sum [cost 10000000000000 (0 - 1) m | m <- machines]

main = "Advent of Code 2024 - Day 13 Results:\n" ++
       "  Part 1 (fewest tokens, <=100 presses): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (with prize offset): " ++ show solvePart2 ++ "\n"
