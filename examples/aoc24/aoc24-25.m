|| Advent of Code 2024 - Day 25: Code Chronicle
|| https://adventofcode.com/2024/day/25
|| The input is a set of lock and key schematics (7 rows x 5 columns). A lock
|| has its top row filled; a key has its bottom row filled. Each is reduced to
|| five column heights. A lock and key fit when, in every column, the two
|| heights sum to at most 5 (no overlap). Part 1 counts the fitting pairs.
|| (Day 25 has no part 2 - it unlocks once the other 49 stars are collected.)
||
|| inputs/day25.txt is seeded with the official example (answer 3); run
|| fetch-inputs.sh for your personal puzzle input.

rawLines = lines (read "examples/aoc24/inputs/day25.txt")

nonEmpty l = l ~= ""

|| split the lines into blocks separated by blank lines
blocks [] = []
blocks ls = grp : blocks (drop 1 rest)
            where
            grp = takewhile nonEmpty ls
            rest = drop (length grp) ls

groups = [g | g <- blocks rawLines; g ~= []]

|| column height = number of '#' in the column, minus the shared full row
colHeight g j = length [1 | row <- g; vec_get (to_vec row) j == '#'] - 1
heights g = [colHeight g j | j <- [0 .. 4]]

isLock g = hd g == "#####"
locks = [heights g | g <- groups; isLock g]
keys = [heights g | g <- groups; ~ isLock g]

|| a lock and key fit if every column pair sums to at most 5
fits l k = and [a + b <= 5 | (a, b) <- zip (l, k)]

solvePart1 = length [1 | l <- locks; k <- keys; fits l k]

main = "Advent of Code 2024 - Day 25 Results:\n" ++
       "  Part 1 (fitting lock/key pairs): " ++ show solvePart1 ++ "\n" ++
       "  (Day 25 has no part 2.)\n"
