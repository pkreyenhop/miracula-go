|| Advent of Code 2025 - Day 2: Gift Shop

solvePart1 input = aoc2_solver input 1
solvePart2 input = aoc2_solver input 2

main = "Advent of Code 2025 - Day 2 Results:\n" ++
       "  Part 1 (Sum of invalid IDs): " ++ show p1Result ++ "\n" ++
       "  Part 2 (Sum of invalid IDs): " ++ show p2Result ++ "\n"
       where
       input = read "inputs/day2.txt"
       p1Result = solvePart1 input
       p2Result = solvePart2 input
