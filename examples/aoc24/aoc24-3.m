|| Advent of Code 2024 - Day 3: Mull It Over
|| https://adventofcode.com/2024/day/3
|| The input is corrupted memory. Part 1: find every valid mul(X,Y)
|| instruction and sum the products. Part 2: do() and don't() instructions
|| enable/disable the muls that follow them.
||
|| inputs/day3.txt is seeded with the official part 1 example (answer 161);
|| day3-example2.txt is the part 2 example (answer 48). The part 2 result for
|| the seeded part 1 example is 161 as well only by coincidence of content -
|| run fetch-inputs.sh for your personal puzzle input.

posFrom n c [] = 0 - 1
posFrom n c (x:xs) = if x == c then n else posFrom (n + 1) c xs

isdig c = posFrom 0 c "0123456789" >= 0

starts [] s = True
starts (x:xs) [] = False
starts (x:xs) (c:cs) = x == c & starts xs cs

|| tryMul "mul(11,8)..." = (88, "...") ; (-1, s) when s does not start with a mul
tryMul s = args (drop 4 s), if starts "mul(" s
         = (0 - 1, s), otherwise
           where
           args t = c1
                    where
                    ds1 = takewhile isdig t
                    r1 = drop (length ds1) t
                    c1 = if ds1 == "" \/ length ds1 > 3 \/ r1 == "" then (0 - 1, s)
                         else if hd r1 ~= ',' then (0 - 1, s) else c2
                    ds2 = takewhile isdig (tl r1)
                    r2 = drop (length ds2) (tl r1)
                    c2 = if ds2 == "" \/ length ds2 > 3 \/ r2 == "" then (0 - 1, s)
                         else if hd r2 ~= ')' then (0 - 1, s)
                         else (numval ds1 * numval ds2, tl r2)

solvePart1 [] = 0
solvePart1 s = if v >= 0 then v + solvePart1 rest else solvePart1 (tl s)
               where
               mr = tryMul s
               v = fst mr
               rest = snd mr

|| scan with an enabled flag toggled by do() / don't()
scan2 on [] = 0
scan2 on s = scan2 True (drop 4 s), if starts "do()" s
           = scan2 False (drop 7 s), if starts "don't()" s
           = (if on then v else 0) + scan2 on rest, if v >= 0
           = scan2 on (tl s), otherwise
             where
             mr = tryMul s
             v = fst mr
             rest = snd mr

solvePart2 s = scan2 True s

main = "Advent of Code 2024 - Day 3 Results:\n" ++
       "  Part 1 (sum of muls): " ++ show p1 ++ "\n" ++
       "  Part 2 (sum of enabled muls): " ++ show p2 ++ "\n"
       where
       input = read "examples/aoc24/inputs/day3.txt"
       p1 = solvePart1 input
       p2 = solvePart2 input
