|| Advent of Code 2024 - Day 15: Warehouse Woes
|| https://adventofcode.com/2024/day/15
|| A robot shoves boxes around a warehouse following a move string. Part 1 is
|| plain Sokoban with 1-wide boxes 'O'. Part 2 doubles the width so each box
|| becomes '[]' spanning two cells, and a vertical push can fan out to several
|| boxes at once. The answer is the sum of GPS coordinates (100*row + col) of
|| every box (its left cell in part 2).
||
|| The grid is stored as a map (cell code -> char) holding walls '#' and boxes
|| ('O', or '[' and ']'); the robot position is tracked separately and the
|| state (grid, pos) is threaded through the strict foldl over the moves.
||
|| inputs/day15.txt is seeded with the large official example (answers
|| 10092 / 9021); run fetch-inputs.sh to replace it with your puzzle input.

fstp (a, b) = a
sndp (a, b) = b
haschar c [] = False
haschar c (x:xs) = if x == c then True else haschar c xs
concatAll [] = []
concatAll (x:xs) = x ++ concatAll xs

rawLines = lines (read "examples/aoc24/inputs/day15.txt")
gridLines = [l | l <- rawLines; haschar '#' l]
moveStr = concatAll [l | l <- rawLines; ~ haschar '#' l;
                         haschar '<' l \/ haschar '>' l \/ haschar '^' l \/ haschar 'v' l]

nrows = length gridLines
ncols = length (hd gridLines)

|| direction (dr, dc) for a move character
dirOf '^' = (0 - 1, 0)
dirOf 'v' = (1, 0)
dirOf '<' = (0, 0 - 1)
dirOf '>' = (0, 1)

|| ---------------------------------------------------------------- Part 1 ----
w1 = ncols
code1 r c = r * w1 + c

|| initial map (floor '.', walls '#', boxes 'O') and robot position
cells1 = [(r, c, ch) | r <- [0 .. nrows - 1]; c <- [0 .. ncols - 1];
                       ch <- [vec_get (to_vec (vec_get (to_vec gridLines) r)) c]]
initGrid1 = foldl ins empty_map cells1
            where ins m (r, c, ch) = h_insert m (code1 r c) (if ch == '@' then '.' else ch)
robot1 = hd [(r, c) | (r, c, ch) <- cells1; ch == '@']

|| first non-box cell scanning from (r,c) in direction (dr,dc)
firstFree1 g r c dr dc = (r, c), if h_lookup g (code1 r c) ~= 'O'
                       = firstFree1 g (r + dr) (c + dc) dr dc, otherwise

|| one move: returns (grid, pos)
step1 (g, (r, c)) mv = state, if npch == '#'
                     = (g, (nr, nc)), if npch == '.'
                     = (g, (r, c)), if ffch == '#'
                     = (h_insert (h_insert g np '.') ff 'O', (nr, nc)), otherwise
                       where
                       state = (g, (r, c))
                       d = dirOf mv
                       dr = fstp d
                       dc = sndp d
                       nr = r + dr
                       nc = c + dc
                       np = code1 nr nc
                       npch = h_lookup g np
                       ffp = firstFree1 g nr nc dr dc
                       ff = code1 (fstp ffp) (sndp ffp)
                       ffch = h_lookup g ff

final1 = foldl step1 (initGrid1, robot1) moveStr
gps1 = sum [100 * r + c | r <- [0 .. nrows - 1]; c <- [0 .. ncols - 1];
                          h_lookup (fstp final1) (code1 r c) == 'O']

|| ---------------------------------------------------------------- Part 2 ----
|| widen: '#'->"##", 'O'->"[]", '.'->"..", '@'->"@." ; boxes are '[' + ']'
widen '#' = "##"
widen 'O' = "[]"
widen '.' = ".."
widen '@' = "@."
wideLines = [concatAll [widen ch | ch <- l] | l <- gridLines]
w2 = ncols * 2
code2 r c = r * w2 + c

cells2 = [(r, c, ch) | r <- [0 .. nrows - 1]; c <- [0 .. w2 - 1];
                       ch <- [vec_get (to_vec (vec_get (to_vec wideLines) r)) c]]
initGrid2 = foldl ins empty_map cells2
            where ins m (r, c, ch) = h_insert m (code2 r c) (if ch == '@' then '.' else ch)
robot2 = hd [(r, c) | (r, c, ch) <- cells2; ch == '@']

get2 g r c = h_lookup g (code2 r c)

|| Horizontal push (dc ~= 0): scan along the row for the first '.'/'#'.
firstFreeH g r c dc = (r, c), if ch ~= '[' & ch ~= ']'
                    = firstFreeH g r (c + dc) dc, otherwise
                      where ch = get2 g r c

|| shift the run of cells between the free spot and the robot by one (horizontal);
|| the cell the robot steps into (cto) is cleared to floor.
shiftH g r cfrom cto dc = h_insert g (code2 r cto) '.', if cfrom == cto
                        = shiftH (h_insert g (code2 r cfrom) (get2 g r (cfrom - dc)))
                                 r (cfrom - dc) cto dc, otherwise

|| Vertical push (dr ~= 0): gather the set of box-left-cells that would move.
|| Returns (canMove, listOfBoxLeftCells). A box is identified by its '[' cell.
|| boxesAbove: given a frontier of columns at row r that the robot/boxes push
|| into, collect all boxes recursively. We work with explicit box coordinates.

orAny [] = False
orAny (b:bs) = b \/ orAny bs
elemPair p [] = False
elemPair (a, b) ((x, y):rest) = (a == x & b == y) \/ elemPair (a, b) rest
eqPair (a, b) (x, y) = a == x & b == y
dedup [] = []
dedup (x:xs) = x : dedup [y | y <- xs; ~ eqPair x y]

|| leftCell of a box occupying column c at row r (c may be '[' or ']')
boxLeft g r c = c, if get2 g r c == '['
              = c - 1, otherwise

|| collect all boxes pushed when moving into cells `fronts` (list of (r,c)) in
|| vertical direction dr; returns (ok, boxlist) where boxlist are (r, leftc).
gatherV g dr fronts seen
    = (True, seen), if fronts == []
    = (False, seen), if anyWall
    = gatherV g dr nextFronts allBoxes, otherwise
      where
      || cells actually occupied by box parts
      boxCells = [(r, c) | (r, c) <- fronts; get2 g r c == '[' \/ get2 g r c == ']']
      anyWall = orAny [get2 g r c == '#' | (r, c) <- fronts]
      newBoxes = dedup [(r, boxLeft g r c) | (r, c) <- boxCells]
      freshBoxes = [bx | bx <- newBoxes; ~ elemPair bx seen]
      allBoxes = seen ++ freshBoxes
      nextFronts = concatAll [[(r + dr, lc), (r + dr, lc + 1)] | (r, lc) <- freshBoxes]

|| apply a vertical push: clear all box cells, then redraw shifted by dr
applyV g dr boxes = redraw cleared
                    where
                    cleared = foldl clr g boxes
                    clr m (r, lc) = h_insert (h_insert m (code2 r lc) '.') (code2 r (lc + 1)) '.'
                    redraw m = foldl drw m boxes
                    drw m (r, lc) = h_insert (h_insert m (code2 (r + dr) lc) '[')
                                             (code2 (r + dr) (lc + 1)) ']'

step2 (g, (r, c)) mv = doHoriz, if dr == 0
                     = doVert, otherwise
                       where
                       d = dirOf mv
                       dr = fstp d
                       dc = sndp d
                       nr = r + dr
                       nc = c + dc
                       npch = get2 g nr nc
                       || --- horizontal ---
                       ffp = firstFreeH g r (c + dc) dc
                       ffc = sndp ffp
                       ffch = get2 g r ffc
                       doHoriz = (g, (r, c)), if npch == '#'
                               = (g, (nr, nc)), if npch == '.'
                               = (g, (r, c)), if ffch == '#'
                               = (shiftH g r ffc (c + dc) dc, (nr, nc)), otherwise
                       || --- vertical ---
                       gv = gatherV g dr [(nr, nc)] []
                       canV = fstp gv
                       boxesV = sndp gv
                       doVert = (g, (r, c)), if npch == '#'
                              = (g, (nr, nc)), if npch == '.'
                              = (g, (r, c)), if ~ canV
                              = (applyV g dr boxesV, (nr, nc)), otherwise

final2 = foldl step2 (initGrid2, robot2) moveStr
gps2 = sum [100 * r + c | r <- [0 .. nrows - 1]; c <- [0 .. w2 - 1];
                          get2 (fstp final2) r c == '[']

main = "Advent of Code 2024 - Day 15 Results:\n" ++
       "  Part 1 (GPS sum, narrow): " ++ show gps1 ++ "\n" ++
       "  Part 2 (GPS sum, wide): " ++ show gps2 ++ "\n"
