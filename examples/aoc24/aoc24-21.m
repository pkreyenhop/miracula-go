|| Advent of Code 2024 - Day 21: Keypad Conundrum
|| https://adventofcode.com/2024/day/21
|| A code is typed on a numeric keypad by a robot, which is directed from a
|| directional keypad by another robot, and so on up a chain of directional
|| keypads that you operate at the top. The complexity of a code is
|| (shortest number of buttons you must press) * (numeric part of the code).
|| Part 1 uses 2 directional keypads in the chain, part 2 uses 25.
||
|| We compute the cost recursively: to type a button we move the arm from its
|| current button to the target (choosing a gap-avoiding shortest path) and
|| press A; the cost of those directional presses is the same problem one level
|| shallower. cost(from, to, depth) on the directional keypad is memoized in a
|| map threaded through the recursion (there are only a few hundred states).
||
|| inputs/day21.txt is seeded with the official example (part 1 answer 126384);
|| run fetch-inputs.sh for your personal puzzle input.

fstp (a, b) = a
sndp (a, b) = b
absn n = if n < 0 then 0 - n else n
posFrom n c [] = 0 - 1
posFrom n c (x:xs) = if x == c then n else posFrom (n + 1) c xs
replicate v 0 = []
replicate v n = v : replicate v (n - 1)
concatAll [] = []
concatAll (x:xs) = x ++ concatAll xs

|| ---- keypad geometry ----------------------------------------------------
|| numeric keypad positions (row, col); gap at (3,0)
numpos '7' = (0,0)
numpos '8' = (0,1)
numpos '9' = (0,2)
numpos '4' = (1,0)
numpos '5' = (1,1)
numpos '6' = (1,2)
numpos '1' = (2,0)
numpos '2' = (2,1)
numpos '3' = (2,2)
numpos '0' = (3,1)
numpos 'A' = (3,2)
numGap = (3,0)

|| directional keypad positions; gap at (0,0)
dirpos '^' = (0,1)
dirpos 'A' = (0,2)
dirpos '<' = (1,0)
dirpos 'v' = (1,1)
dirpos '>' = (1,2)
dirGap = (0,0)

applyMove (r, c) '^' = (r - 1, c)
applyMove (r, c) 'v' = (r + 1, c)
applyMove (r, c) '<' = (r, c - 1)
applyMove (r, c) '>' = (r, c + 1)

eqPos (a, b) (c, d) = a == c & b == d
okPath pos [] gap = ~ eqPos pos gap
okPath pos (m:ms) gap = ~ eqPos pos gap & okPath (applyMove pos m) ms gap

dedup [] = []
dedup (x:xs) = x : dedup [y | y <- xs; y ~= x]

|| the (one or two) gap-avoiding shortest move strings between two positions
pathsBetween fp tp gap = dedup [p | p <- [cand1, cand2]; okPath fp p gap]
                         where
                         fr = fstp fp
                         fc = sndp fp
                         tr = fstp tp
                         tc = sndp tp
                         vch = if tr > fr then 'v' else '^'
                         hch = if tc > fc then '>' else '<'
                         vmoves = replicate vch (absn (tr - fr))
                         hmoves = replicate hch (absn (tc - fc))
                         cand1 = hmoves ++ vmoves
                         cand2 = vmoves ++ hmoves

npaths a b = pathsBetween (numpos a) (numpos b) numGap
dpaths a b = pathsBetween (dirpos a) (dirpos b) dirGap

|| ---- cost tables, built iteratively (no cross-recursion) ----------------
|| tbl k is a map from a directional move (a -> b) to the number of buttons a
|| human must press to make a robot arm perform it, when this keypad sits under
|| k further directional keypads. tbl 0 is a direct human press (path + A).
|| tbl (k+1) is built from tbl k. `tables = iterate step tbl0` computes the
|| whole stack lazily and shares it (a memoised CAF).
ci c = posFrom 0 c "A^v<>"
pkey a b = ci a * 5 + ci b

minList (x:[]) = x
minList (x:xs) = if x < m then x else m
                 where m = minList xs

pairize (x:y:rest) = (x, y) : pairize (y : rest)
pairize xs = []

dirButtons = "A^v<>"
allDpairs = [(a, b) | a <- dirButtons; b <- dirButtons]

|| cost of typing string `s` on a directional keypad, given the table `tbl` for
|| the keypad one level up (sum the per-move costs of A:s). NB `seq` is a
|| builtin and cannot be used as a parameter name.
seqOnTbl s tbl = sum [h_lookup tbl (pkey (fstp pr) (sndp pr)) | pr <- pairize ('A' : s)]

cost0 a b = length (hd (dpaths a b)) + 1
ins0 m ab = h_insert m (pkey (fstp ab) (sndp ab)) (cost0 (fstp ab) (sndp ab))
tbl0 = foldl ins0 empty_map allDpairs

bestMove prev a b = minList [seqOnTbl (p ++ "A") prev | p <- dpaths a b]
insStep prev m ab = h_insert m (pkey (fstp ab) (sndp ab)) (bestMove prev (fstp ab) (sndp ab))
step prev = foldl (insStep prev) empty_map allDpairs

nthT 0 (x:xs) = x
nthT n (x:xs) = nthT (n - 1) xs
tables = iterate step tbl0
tableForLevels levels = nthT (levels - 1) tables

|| a numeric-keypad move expands to directional presses typed on the first
|| dirpad, which has (levels - 1) further dirpads above it
codeCost code levels = sum [minList [seqOnTbl (p ++ "A") tbl | p <- npaths (fstp pr) (sndp pr)]
                            | pr <- pairize ('A' : code)]
                       where tbl = tableForLevels levels

isDig c = posFrom 0 c "0123456789" >= 0
numericPart code = numval [c | c <- code; isDig c]

codes = [l | l <- lines (read "examples/aoc24/inputs/day21.txt"); l ~= ""]
complexity levels code = codeCost code levels * numericPart code

solvePart1 = sum [complexity 2 code | code <- codes]
solvePart2 = sum [complexity 25 code | code <- codes]

main = "Advent of Code 2024 - Day 21 Results:\n" ++
       "  Part 1 (2 directional keypads): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (25 directional keypads): " ++ show solvePart2 ++ "\n"
