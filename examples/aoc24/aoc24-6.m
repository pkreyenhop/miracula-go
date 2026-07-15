|| Advent of Code 2024 - Day 6: Guard Gallivant
|| https://adventofcode.com/2024/day/6
|| A guard walks straight until it hits '#', then turns right; it leaves when
|| it steps off the grid. Part 1: how many distinct cells it visits. Part 2:
|| how many single extra obstacles would trap it in a loop. Loops are detected
|| by revisiting the same (cell, direction) state.
||
|| inputs/day6.txt is seeded with the official example (answers 41 / 6);
|| run fetch-inputs.sh to replace it with your personal puzzle input.

gls = [l | l <- lines (read "examples/aoc24/inputs/day6.txt"); l ~= ""]
grid = to_vec [to_vec l | l <- gls]
nrows = vec_len grid
ncols = vec_len (vec_get grid 0)

cellAt r c = vec_get (vec_get grid r) c

inside r c = r >= 0 & r < nrows & c >= 0 & c < ncols

|| starting cell (the '^') as r * ncols + c
startPos = hd [r * ncols + c | r <- [0 .. nrows - 1]; c <- [0 .. ncols - 1];
                               cellAt r c == '^']

|| direction deltas, 0=up 1=right 2=down 3=left ; turning right is (d+1) mod 4
drOf d = vec_get (to_vec [0 - 1, 0, 1, 0]) d
dcOf d = vec_get (to_vec [0, 1, 0, 0 - 1]) d

|| blocked r c extra : is (r,c) a wall, either original '#' or the extra obstacle
blocked r c extra = (r * ncols + c == extra) \/ (inside r c & cellAt r c == '#')

|| walk collecting the set of visited cell codes; extra = -1 means no extra wall
visitedSet extra = go (startPos / ncols) (startPos mod ncols) 0 empty_set
  where
  go r c d seen = seen2, if ~ inside nr nc
                = go r c ((d + 1) mod 4) seen2, if blocked nr nc extra
                = go nr nc d seen2, otherwise
                  where
                  nr = r + drOf d
                  nc = c + dcOf d
                  seen2 = s_insert seen (r * ncols + c)

visited1 = visitedSet (0 - 1)

|| does adding a wall at `extra` create a loop? track (cell,dir) states.
loopsWith extra = go (startPos / ncols) (startPos mod ncols) 0 empty_set
  where
  go r c d states = False, if ~ inside nr nc
                  = True, if member states key
                  = go r c ((d + 1) mod 4) (s_insert states key), if blocked nr nc extra
                  = go nr nc d (s_insert states key), otherwise
                    where
                    nr = r + drOf d
                    nc = c + dcOf d
                    key = (r * ncols + c) * 4 + d

|| the visited cells as a list (walk the grid, keep cells in the set)
s_toList s = [r * ncols + c | r <- [0 .. nrows - 1]; c <- [0 .. ncols - 1];
                              member s (r * ncols + c)]

visitedCells = s_toList visited1

|| candidate obstacle cells: every visited cell except the start
candidates = [p | p <- visitedCells; p ~= startPos]

solvePart1 = length visitedCells
solvePart2 = length [1 | p <- candidates; loopsWith p]

main = "Advent of Code 2024 - Day 6 Results:\n" ++
       "  Part 1 (visited cells): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (loop obstacles): " ++ show solvePart2 ++ "\n"
