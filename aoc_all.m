|| ============================================================================
|| Advent of Code 2025 - All Days Solved in Miracula (Pure & Go-Helper Hybrid)
|| ============================================================================

|| Day 1
mod100 n = (n mod 100 + 100) mod 100
parseInstruction1 (dir:distStr) = if dir == 'L' then 0 - numval distStr else numval distStr

solveDay1_p1 instructions = countZeros (drop 1 (reverse finalPositions))
  where
    finalPositions = foldl nextPos [50] instructions
    nextPos (h:history) inst = mod100 (h + inst) : h : history
    countZeros []     = 0
    countZeros (x:xs) = if x == 0 then 1 + countZeros xs else countZeros xs

solveDay1_p2 instructions = countZeros (allTicks 50 instructions)
  where
    allTicks currentPos []         = []
    allTicks currentPos (inst:ins) = path ++ allTicks nextPos ins
      where
        nextPos = mod100 (currentPos + inst)
        path    = if inst < 0 then [ mod100 (currentPos - tick) | tick <- [1 .. (0 - inst)] ] else [ mod100 (currentPos + tick) | tick <- [1 .. inst] ]
    countZeros []     = 0
    countZeros (x:xs) = if x == 0 then 1 + countZeros xs else countZeros xs

|| Day 2
aoc2_p1 input = aoc2_solver input 1
aoc2_p2 input = aoc2_solver input 2

|| Day 3
max_list3 (x:xs) = if xs == [] then x else (if x > m then x else m)
                  where
                  m = max_list3 xs
index_of3 val xs = find 0 xs
                  where
                  find idx [] = -1
                  find idx (y:ys) = if y == val then idx else find (idx+1) ys
largest_joltage3 xs = max_left * 10 + max_right
                     where
                     init_xs = take (length xs - 1) xs
                     max_left = max_list3 init_xs
                     first_idx = index_of3 max_left init_xs
                     max_right = max_list3 (drop (first_idx + 1) xs)
greedy3 0 xs = []
greedy3 n xs = max_val : greedy3 (n-1) (drop (max_idx + 1) xs)
              where
              window = take (length xs - n + 1) xs
              max_val = max_list3 window
              max_idx = index_of3 max_val window
digits_to_val3 xs = foldl (\acc. \d. acc * 10 + d) 0 xs
largest_12_joltage3 xs = digits_to_val3 (greedy3 12 xs)
char_to_digit3 c = numval [c]

|| Day 4
repeat_val4 c 0 = []
repeat_val4 c n = c : repeat_val4 c (n-1)
left4 R = '.' : take (length R - 1) R
right4 R = drop 1 R ++ [ '.' ]
zip8_4 [] [] [] [] [] [] [] [] = []
zip8_4 (a:as) (b:bs) (c:cs) (d:ds) (e:es) (f:fs) (g:gs) (h:hs) = (cnt a + cnt b + cnt c + cnt d + cnt e + cnt f + cnt g + cnt h) : zip8_4 as bs cs ds es fs gs hs
  where
  cnt '@' = 1
  cnt _   = 0
zip8_4 as bs cs ds es fs gs hs = []
pickable_row4 prev curr next = zip_cells curr counts
  where
  L1 = left4 prev
  L2 = prev
  L3 = right4 prev
  L4 = left4 curr
  L5 = right4 curr
  L6 = left4 next
  L7 = next
  L8 = right4 next
  counts = zip8_4 L1 L2 L3 L4 L5 L6 L7 L8
  zip_cells [] [] = []
  zip_cells (c:cs) (cnt:cnts) = (c, (c == '@') & (cnt < 4)) : zip_cells cs cnts
  zip_cells cs cnts = []
pickable_grid4 G = zip_rows (empty : G) G (drop 1 G ++ [empty])
  where
  width = length (hd G)
  empty = repeat_val4 '.' width
  zip_rows [] cs ns = []
  zip_rows ps [] ns = []
  zip_rows ps cs [] = []
  zip_rows (p:ps) (c:cs) (n:ns) = pickable_row4 p c n : zip_rows ps cs ns
  zip_rows ps cs ns = []
count_true4 [] = 0
count_true4 ((c, pick):xs) = (if pick then 1 else 0) + count_true4 xs
solvePart4_p1 G = sum (map count_true4 (pickable_grid4 G))
update_row4 [] = []
update_row4 ((c, pick):xs) = (if pick then '.' else c) : update_row4 xs
solvePart4_p2 G = if round_removed == 0 then 0 else round_removed + solvePart4_p2 next_G
               where
               pg = pickable_grid4 G
               round_removed = sum (map count_true4 pg)
               next_G = map update_row4 pg

|| Day 5
split_by5 c [] = [[]]
split_by5 c (x:xs) = if x == c then [] : split_by5 c xs else (x : hd rest) : tl rest
                    where
                    rest = split_by5 c xs
elem5 x [] = False
elem5 x (y:ys) = if x == y then True else elem5 x ys
is_in_range5 val (start, end) = (val >= start) & (val <= end)
any_range5 val [] = False
any_range5 val (r:rs) = if is_in_range5 val r then True else any_range5 val rs
get_start5 (s, e) = s
qsort5 [] = []
qsort5 (x:xs) = qsort5 [y | y <- xs; get_start5 y < get_start5 x] ++ [x] ++ qsort5 [y | y <- xs; get_start5 y >= get_start5 x]
fst5 (a, b) = a
snd5 (a, b) = b
merge5 [] = []
merge5 (x:xs) = if xs == [] then [x] else (if s2 <= e1 + 1 then merge5 ((s1, max e1 (snd5 (hd xs))) : tl xs) else x : merge5 xs)
               where
               s1 = fst5 x
               e1 = snd5 x
               s2 = fst5 (hd xs)
               max a b = if a > b then a else b
range_len5 (start, end) = end - start + 1
not_empty5 l = l ~= ""
not5 True = False
not5 False = True
parse_range5 rangeStr = (numval startStr, numval endStr)
                       where
                       parts = split_by5 '-' rangeStr
                       startStr = hd parts
                       endStr = hd (tl parts)

|| Day 6
product6 [] = 1
product6 (x:xs) = x * product6 xs
apply_op6 op nums = if op == "+" then sum nums else product6 nums
solve_sub6 problem = apply_op6 op nums
                    where
                    op = last problem
                    nums = map numval (init problem)
                    last (x:xs) = if xs == [] then x else last xs
                    init (x:xs) = if xs == [] then [] else x : init xs
zip5_6 [] [] [] [] [] = []
zip5_6 (a:as) (b:bs) (c:cs) (d:ds) (e:es) = [a, b, c, d, e] : zip5_6 as bs cs ds es
zip5_6 as bs cs ds es = []
solvePart6_p1 tokens = sum (map solve_sub6 subproblems)
  where
  ops = [t | t <- tokens; (t == "+") \/ (t == "*")]
  opCount = length ops
  row1 = take opCount tokens
  row2 = take opCount (drop opCount tokens)
  row3 = take opCount (drop (2*opCount) tokens)
  row4 = take opCount (drop (3*opCount) tokens)
  row5 = take opCount (drop (4*opCount) tokens)
  subproblems = zip5_6 row1 row2 row3 row4 row5
repeat_val6 c 0 = []
repeat_val6 c n = c : repeat_val6 c (n-1)
max_val6 [] = 0
max_val6 (x:xs) = if x > m then x else m
                 where
                 m = max_val6 xs
len_show6 x = length (show x)
transpose6 [] = []
transpose6 ([]:xs) = []
transpose6 xss = map hd xss : transpose6 (map tl xss)
not_zero_char6 c = c ~= '0'
rotate_numbers6 arr = reverse (map numval (map (filter not_zero_char6) (transpose6 digits_matrix)))
                     where
                     digits_matrix = [padded x | x <- arr]
                     max_digits = max_val6 (map len_show6 arr)
                     padded x = repeat_val6 '0' (max_digits - length s) ++ s
                                where
                                s = show x
find_dividers6 idx [] = []
find_dividers6 idx (c1 : cs) = if cs == [] then [] else (if (c1 == ' ') & ((c2 == '+') \/ (c2 == '*')) then idx : find_dividers6 (idx+1) cs else find_dividers6 (idx+1) cs)
                              where
                              c2 = hd cs
format_row6 row dividers maxLen = scan 0 row dividers
  where
  scan idx [] divs = if idx >= maxLen then [] else (if is_div then 'x' else '0') : scan (idx+1) [] next_divs
                     where
                     is_div = (divs ~= []) & (idx == hd divs)
                     next_divs = if is_div then tl divs else divs
  scan idx (c:cs) divs = if idx >= maxLen then [] else (if is_div then 'x' else (if c == ' ' then '0' else c)) : scan (idx+1) cs next_divs
                         where
                         is_div = (divs ~= []) & (idx == hd divs)
                         next_divs = if is_div then tl divs else divs
solve_part2_cols6 [] [] = 0
solve_part2_cols6 (op:ops) (col:cols) = (if op == '+' then sum col else product6 col) + solve_part2_cols6 ops cols
solvePart6_p2 allLines = solve_part2_cols6 operations rotated_cols
  where
  numLines = take (length allLines - 1) allLines
  opLine = last allLines
  last (x:xs) = if xs == [] then x else last xs
  dividers = find_dividers6 0 opLine
  maxLen = length opLine
  formatted = [ format_row6 r dividers maxLen | r <- numLines ]
  parsed_grid = map (map numval) (map (\r. split_by5 'x' r) formatted)
  rotated_cols = map rotate_numbers6 (transpose6 parsed_grid)
  operations = [ c | c <- opLine; (c == '+') \/ (c == '*') ]
replace_nl6 '\n' = ' '
replace_nl6 c = c
not_newline6 c = c ~= '\n'

|| Day 7
aoc7_p1 input = aoc7_solver input

|| Day 8
aoc8_p1 input = aoc8_solver input

|| Day 9
aoc9_p1 input = aoc9_solver input

|| Day 10
aoc10_p1 input = aoc10_solver input

|| Day 11
aoc11_p1 input = aoc11_solver input 1
aoc11_p2 input = aoc11_solver input 2


|| master runners
solveDay1 =
  "Day 1 Results:\n" ++
  "  Part 1 (Landing stops on 0): " ++ show (solveDay1_p1 parsedLines) ++ "\n" ++
  "  Part 2 (Total times touching 0): " ++ show (solveDay1_p2 parsedLines) ++ "\n"
  where
  input = read "inputs/day1.txt"
  parsedLines = map parseInstruction1 (lines input)

solveDay2 =
  "Day 2 Results:\n" ++
  "  Part 1 (Sum of invalid IDs): " ++ show (aoc2_p1 input) ++ "\n" ++
  "  Part 2 (Sum of invalid IDs): " ++ show (aoc2_p2 input) ++ "\n"
  where
  input = read "inputs/day2.txt"

solveDay3 =
  "Day 3 Results:\n" ++
  "  Part 1 (Sum of largest joltages): " ++ show (solvePart1 linesList) ++ "\n" ++
  "  Part 2 (Sum of largest 12-digit joltages): " ++ show (solvePart2 linesList) ++ "\n"
  where
  input = read "inputs/day3.txt"
  linesList = map (map char_to_digit3) (lines input)
  solvePart1 lList = sum (map largest_joltage3 lList)
  solvePart2 lList = sum (map largest_12_joltage3 lList)

solveDay4 =
  "Day 4 Results:\n" ++
  "  Part 1 (Pickable rolls): " ++ show (solvePart4_p1 G) ++ "\n" ++
  "  Part 2 (Total removed rolls): " ++ show (solvePart4_p2 G) ++ "\n"
  where
  input = read "inputs/day4.txt"
  G = lines input

solveDay5 =
  "Day 5 Results:\n" ++
  "  Part 1 (IDs in range): " ++ show (solvePart1 ranges ids) ++ "\n" ++
  "  Part 2 (Total points covered): " ++ show (solvePart2 ranges) ++ "\n"
  where
  input = read "inputs/day5.txt"
  allLines = filter not_empty5 (split_by5 '\n' input)
  rangesLines = [l | l <- allLines; elem5 '-' l]
  idsLines = [l | l <- allLines; not5 (elem5 '-' l)]
  ranges = map parse_range5 rangesLines
  ids = map numval idsLines
  solvePart1 rgs idsList = length [x | x <- idsList; any_range5 x rgs]
  solvePart2 rgs = sum (map range_len5 (merge5 (qsort5 rgs)))

solveDay6 =
  "Day 6 Results:\n" ++
  "  Part 1 (Sum of operations): " ++ show (solvePart6_p1 tokens) ++ "\n" ++
  "  Part 2 (Rotated grid sum): " ++ show (solvePart6_p2 allLines) ++ "\n"
  where
  input = read "inputs/day6.txt"
  tokens = filter not_empty5 (split_by5 ' ' (map replace_nl6 input))
  allLines = filter not_empty5 (split_by5 '\n' input)

solveDay7 =
  "Day 7 Results:\n" ++
  "  Part 1 (Total splits): " ++ show (aoc7_p1 input) ++ "\n"
  where
  input = read "inputs/day7.txt"

solveDay8 =
  "Day 8 Results:\n" ++
  "  Part 1 (Circuit size product): " ++ show (aoc8_p1 input) ++ "\n"
  where
  input = read "inputs/day8.txt"

solveDay9 =
  "Day 9 Results:\n" ++
  "  Part 1 (Maximum area): " ++ show (aoc9_p1 input) ++ "\n"
  where
  input = read "inputs/day9.txt"

solveDay10 =
  "Day 10 Results:\n" ++
  "  Part 1 (Shortest combination count): " ++ show (aoc10_p1 input) ++ "\n"
  where
  input = read "inputs/day10.txt"

solveDay11 =
  "Day 11 Results:\n" ++
  "  Part 1 (Paths count you -> out): " ++ show (aoc11_p1 input) ++ "\n" ++
  "  Part 2 (Paths count svr -> out containing dac & fft): " ++ show (aoc11_p2 input) ++ "\n"
  where
  input = read "inputs/day11.txt"

main =
  "=== Advent of Code 2025 Solutions (Miracula) ===\n\n" ++
  solveDay1 ++ "\n" ++
  solveDay2 ++ "\n" ++
  solveDay3 ++ "\n" ++
  solveDay4 ++ "\n" ++
  solveDay5 ++ "\n" ++
  solveDay6 ++ "\n" ++
  solveDay7 ++ "\n" ++
  solveDay8 ++ "\n" ++
  solveDay9 ++ "\n" ++
  solveDay10 ++ "\n" ++
  solveDay11
