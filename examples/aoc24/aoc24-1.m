|| Advent of Code 2024 - Day 1: Historian Hysteria
|| https://adventofcode.com/2024/day/1
|| Two columns of location IDs. Part 1: sort both columns and add up the
|| pairwise absolute differences. Part 2: similarity score - each left value
|| times the number of times it occurs in the right column.
||
|| inputs/day1.txt is seeded with the official example (answers 11 / 31);
|| run fetch-inputs.sh to replace it with your personal puzzle input.

abs1 n = if n < 0 then 0 - n else n

lefts [] = []
lefts (a:b:rest) = a : lefts rest

rights [] = []
rights (a:b:rest) = b : rights rest

|| countmap :: [num] -> map num num, value -> number of occurrences
countmap xs = foldl bump empty_map xs
              where bump m k = h_insert m k (h_lookup_def m k 0 + 1)

solvePart1 nums = sum [abs1 (a - b) | (a, b) <- zip (ls, rs)]
                  where
                  ls = sort_ints (lefts nums)
                  rs = sort_ints (rights nums)

solvePart2 nums = sum [a * h_lookup_def counts a 0 | a <- lefts nums]
                  where counts = countmap (rights nums)

main = "Advent of Code 2024 - Day 1 Results:\n" ++
       "  Part 1 (total distance): " ++ show p1 ++ "\n" ++
       "  Part 2 (similarity score): " ++ show p2 ++ "\n"
       where
       nums = parse_ints (read "examples/aoc24/inputs/day1.txt")
       p1 = solvePart1 nums
       p2 = solvePart2 nums
