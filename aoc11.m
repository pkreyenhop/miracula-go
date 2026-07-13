|| Advent of Code 2025 - Day 11: DAG Path Counting

solvePart1 input = aoc11_solver input 1
solvePart2 input = aoc11_solver input 2

main = "Advent of Code 2025 - Day 11 Results:\n" ++
       "  Part 1 (Paths count you -> out): " ++ show p1Result ++ "\n" ++
       "  Part 2 (Paths count svr -> out containing dac & fft): " ++ show p2Result ++ "\n"
       where
       input = read "inputs/day11.txt"
       p1Result = solvePart1 input
       p2Result = solvePart2 input
