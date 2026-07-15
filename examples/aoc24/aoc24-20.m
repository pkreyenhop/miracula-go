|| Advent of Code 2024 - Day 20: Race Condition
|| https://adventofcode.com/2024/day/20
|| The racetrack is a single winding corridor. A "cheat" lets you pass through
|| walls for a limited number of picoseconds, teleporting between two track
|| cells; it saves (distAlong[end] - distAlong[start] - manhattan) picoseconds.
|| Part 1 allows 2-ps cheats, part 2 allows 20-ps cheats. We count cheats that
|| save at least THRESHOLD picoseconds.
||
|| We walk the corridor once to get each cell's distance from the start, then
|| for every cell scan the diamond of reachable endpoints (a map lookup each),
|| which is far cheaper than comparing all cell pairs.
||
|| inputs/day20.txt is seeded with the official example; with THRESHOLD = 100
|| the example yields 0 / 0 (its best cheat saves 76), so the run also prints
|| the verifiable counts at the example's own thresholds (1 / 285).
||
|| Run fetch-inputs.sh to replace day20.txt with your personal puzzle input.

threshold = 100

gls = [l | l <- lines (read "examples/aoc24/inputs/day20.txt"); l ~= ""]
grid = to_vec [to_vec l | l <- gls]
nrows = vec_len grid
ncols = vec_len (vec_get grid 0)
cellAt r c = vec_get (vec_get grid r) c
inb r c = r >= 0 & r < nrows & c >= 0 & c < ncols
code r c = r * ncols + c
track r c = inb r c & cellAt r c ~= '#'

findCh ch = hd [(r, c) | r <- [0 .. nrows - 1]; c <- [0 .. ncols - 1]; cellAt r c == ch]
startP = findCh 'S'
endP = findCh 'E'
eCode = code (fst endP) (snd endP)

|| walk the single corridor from S to E, tagging each cell with its distance
pathFrom r c pr pc step
    = (r, c, step) : rest
      where
      rest = [], if code r c == eCode
           = pathFrom nr nc r c (step + 1), otherwise
      nxt = hd [(ar, ac) | (ar, ac) <- [(r-1,c),(r+1,c),(r,c-1),(r,c+1)];
                           track ar ac; ~ (ar == pr & ac == pc)]
      nr = fst nxt
      nc = snd nxt

path = pathFrom (fst startP) (snd startP) (0 - 1) (0 - 1) 0
distMap = foldl ins empty_map path
          where ins m (r, c, d) = h_insert m (code r c) d

|| offsets (dr, dc) whose manhattan length is between 2 and mc
absn n = if n < 0 then 0 - n else n
offsets mc = [(dr, dc) | dr <- [(0 - mc) .. mc]; dc <- [(0 - mc) .. mc];
                         absn dr + absn dc >= 2; absn dr + absn dc <= mc]

|| count cheats of reach `mc` saving at least `thr`. The in-bounds guard is
|| essential: without it code(r+dr, c+dc) can wrap past a row edge onto another
|| row's cell and count phantom cheats.
countCheats mc thr = length [1 | (r, c, d) <- path; (dr, dc) <- offsets mc;
                                 inb (r + dr) (c + dc);
                                 dv <- [h_lookup_def distMap (code (r + dr) (c + dc)) (0 - 1)];
                                 dv >= 0; dv - d - (absn dr + absn dc) >= thr]

solvePart1 = countCheats 2 threshold
solvePart2 = countCheats 20 threshold

main = "Advent of Code 2024 - Day 20 Results:\n" ++
       "  Part 1 (2-ps cheats saving >= " ++ show threshold ++ "): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (20-ps cheats saving >= " ++ show threshold ++ "): " ++ show solvePart2 ++ "\n" ++
       "  [example check] 2-ps saving >= 64: " ++ show (countCheats 2 64) ++ " (expect 1)\n" ++
       "  [example check] 20-ps saving >= 50: " ++ show (countCheats 20 50) ++ " (expect 285)\n"
