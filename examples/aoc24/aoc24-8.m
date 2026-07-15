|| Advent of Code 2024 - Day 8: Resonant Collinearity
|| https://adventofcode.com/2024/day/8
|| Antennas share a frequency (a letter or digit). For each pair of same-
|| frequency antennas, part 1 marks the two mirror points just beyond each
|| antenna; part 2 marks every grid point collinear with the pair (including
|| the antennas themselves). Count the distinct marked points.
||
|| inputs/day8.txt is seeded with the official example (answers 14 / 34);
|| run fetch-inputs.sh to replace it with your personal puzzle input.

gls = [l | l <- lines (read "examples/aoc24/inputs/day8.txt"); l ~= ""]
grid = to_vec [to_vec l | l <- gls]
nrows = vec_len grid
ncols = vec_len (vec_get grid 0)
cellAt r c = vec_get (vec_get grid r) c
inside r c = r >= 0 & r < nrows & c >= 0 & c < ncols

|| all antennas as (freqChar, r, c)
antennas = [(cellAt r c, r, c) | r <- [0 .. nrows - 1]; c <- [0 .. ncols - 1];
                                 cellAt r c ~= '.']

freqOf (f, r, c) = f
rOf (f, r, c) = r
cOf (f, r, c) = c

|| ordered pairs of distinct same-frequency antennas
pairs = [(rOf a, cOf a, rOf b, cOf b) | a <- antennas; b <- antennas;
                                        freqOf a == freqOf b;
                                        (rOf a ~= rOf b) \/ (cOf a ~= cOf b)]

code r c = r * ncols + c

|| part 1: the single mirror point beyond b, at 2b - a
antinode1 (ra, ca, rb, cb) = [code r c | r <- [2 * rb - ra]; c <- [2 * cb - ca];
                                          inside r c]

|| part 2: every grid point on the ray from a through b (a, b, b+d, ...)
antinode2 (ra, ca, rb, cb) = walk ra ca
                             where
                             dr = rb - ra
                             dc = cb - ca
                             walk r c = [], if ~ inside r c
                                      = code r c : walk (r + dr) (c + dc), otherwise

toSet codes = foldl s_insert empty_set codes
s_toList s = [code r c | r <- [0 .. nrows - 1]; c <- [0 .. ncols - 1]; member s (code r c)]
countSet codes = length (s_toList (toSet codes))

solvePart1 = countSet (concat [antinode1 p | p <- pairs])
solvePart2 = countSet (concat [antinode2 p | p <- pairs])

main = "Advent of Code 2024 - Day 8 Results:\n" ++
       "  Part 1 (mirror antinodes): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (all collinear antinodes): " ++ show solvePart2 ++ "\n"
