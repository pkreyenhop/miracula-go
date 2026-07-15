|| Advent of Code 2024 - Day 23: LAN Party
|| https://adventofcode.com/2024/day/23
|| An undirected graph of two-letter computers. Part 1 counts the triangles
|| (fully connected triples) that include a computer whose name starts with
|| 't'. Part 2 finds the largest clique and prints its members sorted and
|| comma-joined (the LAN party password), via Bron-Kerbosch.
||
|| Each two-letter name is encoded as an integer val(c1)*26 + val(c2) so it can
|| be compared, sorted, and used as a map/set key.
||
|| inputs/day23.txt is seeded with the official example (answers 7 /
|| "co,de,ka,ta"); run fetch-inputs.sh for your personal puzzle input.

fstp (a, b) = a
sndp (a, b) = b
alphabet = "abcdefghijklmnopqrstuvwxyz"
posFrom n c [] = 0 - 1
posFrom n c (x:xs) = if x == c then n else posFrom (n + 1) c xs
valOf c = posFrom 0 c alphabet
tIdx = valOf 't'

nodeCode s = valOf (hd s) * 26 + valOf (hd (tl s))
decode n = [vec_get (to_vec alphabet) (n / 26), vec_get (to_vec alphabet) (n mod 26)]

edgeLines = [l | l <- lines (read "examples/aoc24/inputs/day23.txt"); l ~= ""]
edges = [(nodeCode (hd parts), nodeCode (hd (tl parts))) | l <- edgeLines;
                                                           parts <- [split "-" l]]

concatE [] = []
concatE ((u, v):es) = (u, v) : (v, u) : concatE es
concatAll [] = []
concatAll (x:xs) = x ++ concatAll xs

|| neighbour set per node (both directions), and the node list
nsetMap = foldl add empty_map (concatE edges)
          where add m (u, v) = h_insert m u (s_insert (h_lookup_def m u empty_set) v)

nset n = h_lookup_def nsetMap n empty_set
adj u v = member (nset u) v

nodeSet = foldl s_insert empty_set (concatAll [[u, v] | (u, v) <- edges])
allNodes = [n | n <- [0 .. 675]; member nodeSet n]
neighList n = [m | m <- allNodes; member (nset n) m]

startsT n = n / 26 == tIdx

|| ------------------------------------------------------------------ Part 1 --
triangles = [(a, b, c) | a <- allNodes; b <- neighList a; a < b;
                         c <- neighList b; b < c; adj a c]
solvePart1 = length [1 | (a, b, c) <- triangles; startsT a \/ startsT b \/ startsT c]

|| ------------------------------------------------------------------ Part 2 --
interS lst n = [x | x <- lst; member (nset n) x]
remove v lst = [x | x <- lst; x ~= v]
longer a b = if length a >= length b then a else b

|| Enumerate every clique exactly once by only ever extending with a
|| higher-numbered vertex adjacent to all current members (p already holds the
|| vertices adjacent to everything in r). A single self-recursive function
|| sidesteps the interpreter's mutual-recursion and accumulator bugs; the max
|| clique is then the longest one produced.
cliques r p = r : concatAll [cliques (r ++ [v]) [w | w <- p; w > v; adj v w] | v <- p]

allCliques = cliques [] allNodes
maxClique = foldl longer [] allCliques

joinComma [] = ""
joinComma (x:[]) = x
joinComma (x:xs) = x ++ "," ++ joinComma xs
solvePart2 = joinComma [decode n | n <- sort_ints maxClique]

main = "Advent of Code 2024 - Day 23 Results:\n" ++
       "  Part 1 (triangles with a 't' computer): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (LAN party password): " ++ solvePart2 ++ "\n"
