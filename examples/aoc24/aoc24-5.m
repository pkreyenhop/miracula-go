|| Advent of Code 2024 - Day 5: Print Queue
|| https://adventofcode.com/2024/day/5
|| Rules "a|b" say page a must be printed before page b. Part 1: sum the
|| middle pages of the correctly ordered updates. Part 2: reorder the
|| incorrect updates by the rules and sum their middle pages instead.
||
|| inputs/day5.txt is seeded with the official example (answers 143 / 123);
|| run fetch-inputs.sh to replace it with your personal puzzle input.

haschar c [] = False
haschar c (x:xs) = if x == c then True else haschar c xs

alllines = [l | l <- lines (read "examples/aoc24/inputs/day5.txt"); l ~= ""]

|| rules as a set of before*1000+after keys (pages are < 1000)
rulekey (a:b:rest) = a * 1000 + b
rules = foldl s_insert empty_set
              [rulekey (parse_ints l) | l <- alllines; haschar '|' l]

updates = [parse_ints l | l <- alllines; haschar ',' l]

|| ordered when no later page must come before an earlier one
orderedp [] = True
orderedp (x:xs) = and [~ member rules (y * 1000 + x) | y <- xs] & orderedp xs

byRules a b = 0 - 1, if member rules (a * 1000 + b)
            = 1, if member rules (b * 1000 + a)
            = 0, otherwise

middle xs = vec_get (to_vec xs) (length xs / 2)

solvePart1 = sum [middle u | u <- updates; orderedp u]

solvePart2 = sum [middle (sort_by byRules u) | u <- updates; ~ orderedp u]

main = "Advent of Code 2024 - Day 5 Results:\n" ++
       "  Part 1 (middle sum of ordered updates): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (middle sum of reordered updates): " ++ show solvePart2 ++ "\n"
