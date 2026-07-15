|| Advent of Code 2024 - Day 7: Bridge Repair
|| https://adventofcode.com/2024/day/7
|| Each line "target: a b c ..." asks whether operators inserted left-to-right
|| can make the numbers equal the target. Part 1 allows + and *; part 2 also
|| allows || (concatenation). Sum the targets of the solvable equations.
||
|| inputs/day7.txt is seeded with the official example (answers 3749 / 11387);
|| run fetch-inputs.sh to replace it with your personal puzzle input.

orl [] = False
orl (b:bs) = b \/ orl bs

|| concat a b : digits of b appended to a, e.g. 12 `cat` 345 = 12345
pow10 n = if n < 10 then 10 else 10 * pow10 (n / 10)
cat a b = a * pow10 b + b

|| can we reach target from (acc, rest) using the given ops? prune when acc>target
reach withCat target acc [] = acc == target
reach withCat target acc (n:ns)
    = False, if acc > target
    = reach withCat target (acc + n) ns
      \/ reach withCat target (acc * n) ns
      \/ (withCat & reach withCat target (cat acc n) ns), otherwise

solvable withCat (t:ns) = reach withCat t (hd ns) (tl ns)

equations = [parse_ints l | l <- lines (read "examples/aoc24/inputs/day7.txt"); l ~= ""]

headOf (t:ns) = t

solvePart1 = sum [headOf e | e <- equations; solvable False e]
solvePart2 = sum [headOf e | e <- equations; solvable True e]

main = "Advent of Code 2024 - Day 7 Results:\n" ++
       "  Part 1 (+ and * only): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (with concatenation): " ++ show solvePart2 ++ "\n"
