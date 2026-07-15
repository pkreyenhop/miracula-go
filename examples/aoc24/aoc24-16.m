|| Advent of Code 2024 - Day 16: Reindeer Maze
|| https://adventofcode.com/2024/day/16
|| The reindeer starts at S facing east; a forward step costs 1 and a 90-degree
|| turn costs 1000. Part 1 is the cheapest score to reach E. Part 2 counts the
|| tiles that lie on at least one best path.
||
|| We run Dijkstra over (cell, direction) states with a sorted-list frontier
|| (no native priority queue) and a threaded distance map. For part 2 we also
|| run Dijkstra backwards from E; a state is on a best path when its forward
|| distance plus its backward distance equals the best score.
||
|| inputs/day16.txt is seeded with the first official example (answers
|| 7036 / 45); day16-example2.txt is the second (11048 / 64). Run
|| fetch-inputs.sh to replace day16.txt with your personal puzzle input.

big = 1000000000

gls = [l | l <- lines (read "examples/aoc24/inputs/day16.txt"); l ~= ""]
grid = to_vec [to_vec l | l <- gls]
nrows = vec_len grid
ncols = vec_len (vec_get grid 0)
cellAt r c = vec_get (vec_get grid r) c
passable r c = cellAt r c ~= '#'

findCh ch = hd [(r, c) | r <- [0 .. nrows - 1]; c <- [0 .. ncols - 1]; cellAt r c == ch]
startP = findCh 'S'
endP = findCh 'E'

|| state = (r*ncols + c)*4 + dir ; dir 0=E 1=S 2=W 3=N
enc r c d = (r * ncols + c) * 4 + d
stDir s = s mod 4
stR s = (s / 4) / ncols
stC s = (s / 4) mod ncols
drOf d = vec_get (to_vec [0, 1, 0, 0 - 1]) d
dcOf d = vec_get (to_vec [1, 0, 0 - 1, 0]) d

|| forward transitions: two turns (cost 1000) and a step ahead (cost 1)
transF s = turns ++ fwd
           where
           r = stR s
           c = stC s
           d = stDir s
           turns = [(1000, enc r c ((d + 1) mod 4)), (1000, enc r c ((d + 3) mod 4))]
           nr = r + drOf d
           nc = c + dcOf d
           fwd = [(1, enc nr nc d) | passable nr nc]

|| backward transitions: turns are symmetric; the step comes from behind
transB s = turns ++ back
           where
           r = stR s
           c = stC s
           d = stDir s
           turns = [(1000, enc r c ((d + 1) mod 4)), (1000, enc r c ((d + 3) mod 4))]
           pr = r - drOf d
           pc = c - dcOf d
           back = [(1, enc pr pc d) | passable pr pc]

|| Dijkstra from the given start states, using the native priority queue and
|| tuple-destructuring `let`; returns the settled distance map.
seedPQ q s = pq_push q 0 s
seedDist m s = h_insert m s 0
pushEdge q e = pq_push q (fst e) (snd e)
relaxEdge m e = h_insert m (snd e) (fst e)

dijkstra trans starts = loop (foldl seedPQ pq_empty starts) (foldl seedDist empty_map starts)
                        where
                        loop pq dist
                            = dist, if pq_null pq
                            = loop rest dist, if cost > h_lookup_def dist st big
                            = loop (foldl pushEdge rest relaxed) (foldl relaxEdge dist relaxed), otherwise
                              where
                              (cost, st, rest) = pq_pop pq
                              relaxed = [(cost + w, ns) | (w, ns) <- trans st;
                                                          cost + w < h_lookup_def dist ns big]

distF = dijkstra transF [enc (fst startP) (snd startP) 0]
endStates = [enc (fst endP) (snd endP) d | d <- [0 .. 3]]
best = foldl mn big [h_lookup_def distF s big | s <- endStates]
       where mn a b = if b < a then b else a

distB = dijkstra transB endStates

orAny [] = False
orAny (b:bs) = b \/ orAny bs

|| a tile is on a best path if some direction state has fwd + back == best
onBest r c = orAny [h_lookup_def distF s big + h_lookup_def distB s big == best
                    | d <- [0 .. 3]; s <- [enc r c d]]

solvePart1 = best
solvePart2 = length [1 | r <- [0 .. nrows - 1]; c <- [0 .. ncols - 1];
                        passable r c; onBest r c]

main = "Advent of Code 2024 - Day 16 Results:\n" ++
       "  Part 1 (lowest score): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (tiles on best paths): " ++ show solvePart2 ++ "\n"
