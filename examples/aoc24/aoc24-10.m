|| Advent of Code 2024 - Day 10: Hoof It
|| https://adventofcode.com/2024/day/10
|| A topographic map of heights 0..9. A trail steps +1 in height between
|| orthogonal neighbours. Part 1: sum over trailheads (height 0) of the number
|| of distinct 9s reachable. Part 2: sum of the number of distinct trails.
||
|| inputs/day10.txt is seeded with the official example (answers 36 / 81);
|| run fetch-inputs.sh to replace it with your personal puzzle input.

posFrom n c [] = 0 - 1
posFrom n c (x:xs) = if x == c then n else posFrom (n + 1) c xs
digitv c = posFrom 0 c "0123456789"

gls = [l | l <- lines (read "examples/aoc24/inputs/day10.txt"); l ~= ""]
grid = to_vec [to_vec [digitv c | c <- l] | l <- gls]
nrows = vec_len grid
ncols = vec_len (vec_get grid 0)
hAt r c = vec_get (vec_get grid r) c
inside r c = r >= 0 & r < nrows & c >= 0 & c < ncols

code r c = r * ncols + c
neighbours r c = [(nr, nc) | (nr, nc) <- [(r-1,c),(r+1,c),(r,c-1),(r,c+1)]; inside nr nc]

setElems s = [code r c | r <- [0 .. nrows - 1]; c <- [0 .. ncols - 1]; member s (code r c)]

|| reachable set of 9-cells from (r,c) at height h (part 1: distinct summits)
reach r c = foldl s_insert empty_set [code r c], if hAt r c == 9
          = foldl unionInto empty_set nexts, otherwise
            where
            h = hAt r c
            nexts = [reach nr nc | (nr, nc) <- neighbours r c; hAt nr nc == h + 1]
            unionInto acc s = foldl s_insert acc (setElems s)

|| rating: number of distinct trails from (r,c) to a 9 (part 2)
rating r c = 1, if hAt r c == 9
           = sum [rating nr nc | (nr, nc) <- neighbours r c; hAt nr nc == h + 1], otherwise
             where h = hAt r c

heads = [(r, c) | r <- [0 .. nrows - 1]; c <- [0 .. ncols - 1]; hAt r c == 0]

solvePart1 = sum [length (setElems (reach r c)) | (r, c) <- heads]
solvePart2 = sum [rating r c | (r, c) <- heads]

main = "Advent of Code 2024 - Day 10 Results:\n" ++
       "  Part 1 (sum of trailhead scores): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (sum of trailhead ratings): " ++ show solvePart2 ++ "\n"
