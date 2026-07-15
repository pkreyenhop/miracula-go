|| Advent of Code 2024 - Day 4: Ceres Search
|| https://adventofcode.com/2024/day/4
|| A letter grid word search. Part 1: count every occurrence of "XMAS" in
|| all 8 directions. Part 2: count X-shaped pairs of "MAS" crossing at the A.
||
|| inputs/day4.txt is seeded with the official example (answers 18 / 9);
|| run fetch-inputs.sh to replace it with your personal puzzle input.

gls = [l | l <- lines (read "examples/aoc24/inputs/day4.txt"); l ~= ""]
grid = to_vec [to_vec l | l <- gls]
nrows = vec_len grid
ncols = vec_len (vec_get grid 0)

|| att r c: character at (r,c), '.' outside the grid
att r c = if r < 0 \/ r >= nrows \/ c < 0 \/ c >= ncols then '.'
          else vec_get (vec_get grid r) c

dirs = [(dr, dc) | dr <- [0 - 1, 0, 1]; dc <- [0 - 1, 0, 1]; ~(dr == 0 & dc == 0)]

xmasFrom r c (dr, dc) = att r c == 'X'
                        & att (r + dr) (c + dc) == 'M'
                        & att (r + 2 * dr) (c + 2 * dc) == 'A'
                        & att (r + 3 * dr) (c + 3 * dc) == 'S'

solvePart1 = length [1 | r <- [0 .. nrows - 1]; c <- [0 .. ncols - 1];
                         d <- dirs; xmasFrom r c d]

|| an X-MAS: 'A' centre with "MAS" or "SAM" on both diagonals
masDiag r1 c1 r2 c2 = (att r1 c1 == 'M' & att r2 c2 == 'S')
                      \/ (att r1 c1 == 'S' & att r2 c2 == 'M')

xmasAt r c = att r c == 'A'
             & masDiag (r - 1) (c - 1) (r + 1) (c + 1)
             & masDiag (r - 1) (c + 1) (r + 1) (c - 1)

solvePart2 = length [1 | r <- [0 .. nrows - 1]; c <- [0 .. ncols - 1]; xmasAt r c]

main = "Advent of Code 2024 - Day 4 Results:\n" ++
       "  Part 1 (XMAS count): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (X-MAS count): " ++ show solvePart2 ++ "\n"
