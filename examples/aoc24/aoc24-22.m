|| Advent of Code 2024 - Day 22: Monkey Market
|| https://adventofcode.com/2024/day/22
|| Each buyer's secret evolves by a little xor/shift PRNG. Part 1 sums every
|| buyer's 2000th secret. Part 2: the price is the secret's last digit; find
|| the four-consecutive-price-change sequence that, summed over all buyers
|| (first occurrence per buyer), yields the most bananas.
||
|| "mix" is the native bitwise xor.
||
|| inputs/day22.txt is seeded with the part 1 example (answer 37327623);
|| day22-example2.txt is the part 2 example (answer 23). Run fetch-inputs.sh
|| for your personal puzzle input.

prune x = x mod 16777216
step s = s3
         where
         s1 = prune (xor s (s * 64))
         s2 = prune (xor s1 (s1 / 32))
         s3 = prune (xor s2 (s2 * 2048))

seeds = parse_ints (read "examples/aoc24/inputs/day22.txt")

|| 2000th secret of one buyer
secret2000 seed = hd (drop 2000 (iterate step seed))
solvePart1 = sum [secret2000 s | s <- seeds]

|| first-occurrence (changeKey, price) pairs for one buyer
firstOccs seed = collect 0 empty_set
                 where
                 prs = to_vec [x mod 10 | x <- take 2001 (iterate step seed)]
                 chg j = vec_get prs (j + 1) - vec_get prs j
                 keyAt i = (((c0 + 9) * 19 + (c1 + 9)) * 19 + (c2 + 9)) * 19 + (c3 + 9)
                           where
                           c0 = chg i
                           c1 = chg (i + 1)
                           c2 = chg (i + 2)
                           c3 = chg (i + 3)
                 collect i seen = [], if i > 1996
                                = collect (i + 1) seen, if member seen k
                                = (k, vec_get prs (i + 4)) : collect (i + 1) (s_insert seen k), otherwise
                                  where k = keyAt i

allPairs = concat [firstOccs s | s <- seeds]

|| sum the bananas per change-sequence key, then take the best
sortedPairs = sort_by cmp allPairs
              where cmp a b = fst a - fst b

groupSums [] = []
groupSums (p:ps) = sum [snd q | q <- grp] : groupSums (drop (length grp) (p:ps))
                   where grp = takewhile (\q. fst q == fst p) (p:ps)

maxOf [] = 0
maxOf (x:xs) = foldl mx x xs
               where mx a b = if b > a then b else a

solvePart2 = maxOf (groupSums sortedPairs)

main = "Advent of Code 2024 - Day 22 Results:\n" ++
       "  Part 1 (sum of 2000th secrets): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (most bananas): " ++ show solvePart2 ++ "\n"
