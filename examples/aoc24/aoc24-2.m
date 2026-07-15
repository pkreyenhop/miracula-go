|| Advent of Code 2024 - Day 2: Red-Nosed Reports
|| https://adventofcode.com/2024/day/2
|| Each line is a report of levels. A report is safe when the levels are
|| strictly monotonic with adjacent gaps of 1..3. Part 1 counts safe reports;
|| part 2 also allows removing one single level (the "Problem Dampener").
||
|| inputs/day2.txt is seeded with the official example (answers 2 / 4);
|| run fetch-inputs.sh to replace it with your personal puzzle input.

andl [] = True
andl (b:bs) = b & andl bs

orl [] = False
orl (b:bs) = b \/ orl bs

diffs (a:b:rest) = (b - a) : diffs (b : rest)
diffs xs = []

gapsOk ds = andl [1 <= d & d <= 3 | d <- ds]

safe xs = gapsOk ds \/ gapsOk [0 - d | d <- ds]
          where ds = diffs xs

dropAt i xs = take i xs ++ drop (i + 1) xs

safeDamped xs = safe xs \/ orl [safe (dropAt i xs) | i <- [0 .. length xs - 1]]

main = "Advent of Code 2024 - Day 2 Results:\n" ++
       "  Part 1 (safe reports): " ++ show p1 ++ "\n" ++
       "  Part 2 (safe with dampener): " ++ show p2 ++ "\n"
       where
       reports = [parse_ints l | l <- lines (read "examples/aoc24/inputs/day2.txt"); l ~= ""]
       p1 = length (filter safe reports)
       p2 = length (filter safeDamped reports)
