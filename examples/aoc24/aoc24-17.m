|| Advent of Code 2024 - Day 17: Chronospatial Computer
|| https://adventofcode.com/2024/day/17
|| A 3-bit virtual machine with registers A, B, C and eight opcodes. Part 1
|| runs the program and prints its comma-separated output. Part 2 finds the
|| lowest initial A that makes the program output its own source, built one
|| octal digit at a time from the last output backwards (each loop consumes
|| three bits of A).
||
|| Miracula has no bitwise operators, so XOR is implemented on the bits.
||
|| inputs/day17.txt is seeded with the official part 1 example (output
|| 4,6,3,5,6,3,5,2,1,0); day17-example2.txt is the part 2 example (answer
|| 117440). Run fetch-inputs.sh for your personal puzzle input.

fstp (a, b) = a
sndp (a, b) = b

|| bitwise xor of two non-negative integers
xorb a b = 0, if a == 0 & b == 0
         = (a mod 2 + b mod 2) mod 2 + 2 * xorb (a / 2) (b / 2), otherwise

pow2 0 = 1
pow2 n = 2 * pow2 (n - 1)

nums = parse_ints (read "examples/aoc24/inputs/day17.txt")
regA = hd nums
regB = hd (tl nums)
regC = hd (tl (tl nums))
prog = tl (tl (tl nums))
progVec = to_vec prog
plen = vec_len progVec

|| run the machine from (a,b,c); returns the list of outputs.
|| state is threaded explicitly and outputs are consed (then reversed).
run a0 b0 c0 = reverse (go a0 b0 c0 0 [])
               where
               go a b c ip out
                   = out, if ip >= plen
                   = go2, otherwise
                     where
                     opcode = vec_get progVec ip
                     lit = vec_get progVec (ip + 1)
                     combo = lit, if lit <= 3
                           = a, if lit == 4
                           = b, if lit == 5
                           = c, if lit == 6
                           = 0, otherwise
                     go2 = go (a / pow2 combo) b c (ip + 2) out, if opcode == 0
                         = go a (xorb b lit) c (ip + 2) out, if opcode == 1
                         = go a (combo mod 8) c (ip + 2) out, if opcode == 2
                         = go a b c lit out, if opcode == 3 & a ~= 0
                         = go a b c (ip + 2) out, if opcode == 3
                         = go a (xorb b c) c (ip + 2) out, if opcode == 4
                         = go a b c (ip + 2) ((combo mod 8) : out), if opcode == 5
                         = go a (a / pow2 combo) c (ip + 2) out, if opcode == 6
                         = go a b (a / pow2 combo) (ip + 2) out, otherwise

joinComma [] = ""
joinComma (x:[]) = show x
joinComma (x:xs) = show x ++ "," ++ joinComma xs

solvePart1 = joinComma (run regA regB regC)

|| Part 2: candidate A values built high octal digit first. `cands` holds all
|| A prefixes whose run reproduces the program's last `k` numbers; extend by
|| one octal digit and keep those matching one more number, until full length.
suffixFrom k = drop k prog
eqList [] [] = True
eqList (x:xs) (y:ys) = x == y & eqList xs ys
eqList xs ys = False

extend cands k = [8 * a + d | a <- cands; d <- [0 .. 7];
                              eqList (run (8 * a + d) 0 0) (suffixFrom k)]

|| fold from position plen-1 down to 0, each step prepends one output number
buildAll = go (plen - 1) [0]
           where
           go k cands = cands, if k < 0
                      = go (k - 1) (extend cands k), otherwise

minOf [] = 0 - 1
minOf (x:xs) = foldl mn x xs
               where mn a b = if b < a then b else a

solvePart2 = minOf buildAll

main = "Advent of Code 2024 - Day 17 Results:\n" ++
       "  Part 1 (program output): " ++ solvePart1 ++ "\n" ++
       "  Part 2 (lowest quine A): " ++ show solvePart2 ++ "\n"
