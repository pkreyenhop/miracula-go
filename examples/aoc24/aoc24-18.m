|| Advent of Code 2024 - Day 18: RAM Run
|| https://adventofcode.com/2024/day/18
|| Bytes fall onto a grid, corrupting cells. Part 1: after the first TAKE1
|| bytes, the shortest path (BFS) from the top-left to the bottom-right corner.
|| Part 2: the coordinates of the first byte that seals off the exit entirely,
|| found by binary search over how many bytes have fallen.
||
|| BFS builds a distance map layer by layer (threading a map, not a lazy
|| integer counter, which is unreliable in this interpreter).
||
|| IMPORTANT: the seeded example uses a 7x7 grid with TAKE1 = 12 (answers
|| 22 and "6,1"). For your real puzzle input set SIZE = 71 and TAKE1 = 1024.
||
|| inputs/day18.txt is seeded with the official example; run fetch-inputs.sh
|| to replace it with your personal puzzle input (and switch SIZE/TAKE1).

size = 7
take1 = 12

|| byte coordinates in fall order, as (x, y); code = y*size + x
byteNums = parse_ints (read "examples/aoc24/inputs/day18.txt")
pairUp [] = []
pairUp (x:y:rest) = (x, y) : pairUp rest
bytes = pairUp byteNums
codeOf (x, y) = y * size + x
byteCodes = [codeOf b | b <- bytes]

target = (size - 1) * size + (size - 1)

inb x y = x >= 0 & x < size & y >= 0 & y < size
nbrsOf p = [ny * size + nx | (nx, ny) <- [(x-1,y),(x+1,y),(x,y-1),(x,y+1)]; inb nx ny]
           where
           x = p mod size
           y = p / size

dedup s xs = go xs empty_set
             where
             go [] seen = []
             go (v:vs) seen = go vs seen, if member seen v \/ member s v
                            = v : go vs (s_insert seen v), otherwise

|| layered BFS from cell 0; returns a map cell -> distance
bfs blocked = go [0] (h_insert empty_map 0 0)
              where
              go [] dist = dist
              go frontier dist = go nbrs dist2
                                 where
                                 d = h_lookup dist (hd frontier)
                                 nbrs = dedup blocked [n | cell <- frontier;
                                                           n <- nbrsOf cell;
                                                           ~ inDist n]
                                 inDist n = h_lookup_def dist n (0 - 1) >= 0
                                 dist2 = foldl put dist nbrs
                                 put m n = h_insert m n (d + 1)

blockedK k = foldl s_insert empty_set (take k byteCodes)

distToTarget k = h_lookup_def (bfs (blockedK k)) target (0 - 1)

solvePart1 = distToTarget take1

|| Part 2: smallest k (>= take1) for which the target is unreachable; the
|| answer is the k-th byte. Binary search on k in [take1, #bytes].
nth n xs = hd (drop n xs)
codeStr (x, y) = show x ++ "," ++ show y
search lo hi = codeStr (nth (lo - 1) bytes), if lo >= hi
             = search (mid + 1) hi, if reachable
             = search lo mid, otherwise
               where
               mid = (lo + hi) / 2
               reachable = distToTarget mid >= 0

solvePart2 = search (take1 + 1) (length bytes)

main = "Advent of Code 2024 - Day 18 Results:\n" ++
       "  Part 1 (shortest path after " ++ show take1 ++ " bytes): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (first blocking byte): " ++ solvePart2 ++ "\n"
