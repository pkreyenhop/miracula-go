|| Advent of Code 2024 - Day 12: Garden Groups
|| https://adventofcode.com/2024/day/12
|| Same-letter cells joined orthogonally form a region. Part 1 prices each
|| region area * perimeter; part 2 prices area * number of straight sides.
|| The number of sides equals the number of corners, which we count locally
|| at each cell (convex + concave corners).
||
|| inputs/day12.txt is seeded with the official example (answers 1930 / 1206);
|| run fetch-inputs.sh to replace it with your personal puzzle input.

fstp (a, b) = a
sndp (a, b) = b

gls = [l | l <- lines (read "examples/aoc24/inputs/day12.txt"); l ~= ""]
grid = to_vec [to_vec l | l <- gls]
nrows = vec_len grid
ncols = vec_len (vec_get grid 0)
cellAt r c = vec_get (vec_get grid r) c
inside r c = r >= 0 & r < nrows & c >= 0 & c < ncols
code r c = r * ncols + c

|| same-plant test that is safe outside the grid (returns False)
same r c ch = inside r c & cellAt r c == ch

|| flood fill from (r,c): returns (visitedSet, cellsList) for the whole region
flood r c ch seen = go [(r, c)] seen []
                    where
                    go [] sn acc = (sn, acc)
                    go ((cr, cc):stack) sn acc
                        = go stack sn acc, if member sn (code cr cc)
                        = go (nbrs ++ stack) (s_insert sn (code cr cc)) ((cr, cc):acc), otherwise
                          where
                          nbrs = [(nr, nc) | (nr, nc) <- [(cr-1,cc),(cr+1,cc),(cr,cc-1),(cr,cc+1)];
                                             same nr nc ch; ~ member sn (code nr nc)]

|| perimeter contribution of one cell = number of edges facing a different plant
cellPerim cr cc ch = length [1 | (nr, nc) <- [(cr-1,cc),(cr+1,cc),(cr,cc-1),(cr,cc+1)];
                                 ~ same nr nc ch]

b cond = if cond then 1 else 0

|| corners at one cell: a convex corner where two perpendicular neighbours
|| differ, and a concave corner where both match but the diagonal differs
cellCorners cr cc ch = convex + concave
                       where
                       up = same (cr-1) cc ch
                       dn = same (cr+1) cc ch
                       lf = same cr (cc-1) ch
                       rt = same cr (cc+1) ch
                       ul = same (cr-1) (cc-1) ch
                       ur = same (cr-1) (cc+1) ch
                       dl = same (cr+1) (cc-1) ch
                       dr = same (cr+1) (cc+1) ch
                       convex = b (~up & ~lf) + b (~up & ~rt) + b (~dn & ~lf) + b (~dn & ~rt)
                       concave = b (up & lf & ~ul) + b (up & rt & ~ur)
                                 + b (dn & lf & ~dl) + b (dn & rt & ~dr)

|| walk every cell; when it starts a new region, flood it and record
|| (area, perimeter, sides) as a list element. (A list accumulator is used
|| rather than a hand-rolled integer accumulator, which triggers a lazy
|| thunk-blowup bug in this interpreter; we sum the list afterwards.)
regions = go 0 0 empty_set []
          where
          go r c seen acc
              = acc, if r >= nrows
              = go (r + 1) 0 seen acc, if c >= ncols
              = go r (c + 1) seen acc, if member seen (code r c)
              = go r (c + 1) seen2 ((area, perim, sides) : acc), otherwise
                where
                ch = cellAt r c
                fr = flood r c ch seen
                seen2 = fstp fr
                cells = sndp fr
                area = length cells
                perim = sum [cellPerim cr cc ch | (cr, cc) <- cells]
                sides = sum [cellCorners cr cc ch | (cr, cc) <- cells]

fst3 (a, b, c) = a
snd3 (a, b, c) = b
thd3 (a, b, c) = c

solvePart1 = sum [fst3 rg * snd3 rg | rg <- regions]
solvePart2 = sum [fst3 rg * thd3 rg | rg <- regions]

main = "Advent of Code 2024 - Day 12 Results:\n" ++
       "  Part 1 (area * perimeter): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (area * sides): " ++ show solvePart2 ++ "\n"
