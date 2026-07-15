|| Advent of Code 2024 - Day 24: Crossed Wires
|| https://adventofcode.com/2024/day/24
|| A network of boolean gates. Part 1 evaluates the circuit and reads the
|| z-wires (z00 = least significant) as a binary number. We evaluate by
|| fixpoint: repeatedly sweep the gates, computing any whose inputs are known,
|| until every output is set.
||
|| Part 2 (find the four pairs of swapped gates that break a ripple-carry
|| adder) is a structural puzzle specific to the real 45-bit adder input; the
|| example circuit is not an adder, so there is no example to verify against
|| and it is intentionally left unsolved here. See README.md.
||
|| inputs/day24.txt is seeded with the larger official example (part 1
|| answer 2024); run fetch-inputs.sh for your personal puzzle input.

pow2 0 = 1
pow2 n = 2 * pow2 (n - 1)
haschar c [] = False
haschar c (x:xs) = if x == c then True else haschar c xs
nth n xs = hd (drop n xs)

rawLines = lines (read "examples/aoc24/inputs/day24.txt")
initLines = [l | l <- rawLines; haschar ':' l]
gateLines = [l | l <- rawLines; haschar '>' l]

|| initial wire values as a map (wire -> 0/1)
initVals = foldl ins empty_map initLines
           where
           ins m l = h_insert m (hd parts) (numval (hd (tl parts)))
                     where parts = split ": " l

|| gates as (a, op, b, out)
words4 l = (nth 0 ws, nth 1 ws, nth 2 ws, nth 4 ws)
           where ws = split " " l
gates = [words4 l | l <- gateLines]

known w vals = h_lookup_def vals w (0 - 1) >= 0
applyOp op a b = a * b, if op == "AND"
               = (if a + b > 0 then 1 else 0), if op == "OR"
               = (a + b) mod 2, otherwise

outOf (a, op, b, out) = out
tryGate vals (a, op, b, out)
    = h_insert vals out (applyOp op (h_lookup vals a) (h_lookup vals b)),
        if known a vals & known b vals & ~ known out vals
    = vals, otherwise
onePass vals = foldl tryGate vals gates

fixpoint vals = vals, if kc == length gates
              = fixpoint (onePass vals), otherwise
                where kc = length [1 | g <- gates; known (outOf g) vals]

finalVals = fixpoint initVals

|| assemble the z-wires (z00 is bit 0) into a number
zGates = [out | g <- gates; out <- [outOf g]; hd out == 'z']
zBit w = h_lookup finalVals w * pow2 (numval (tl w))
solvePart1 = sum [zBit w | w <- zGates]

main = "Advent of Code 2024 - Day 24 Results:\n" ++
       "  Part 1 (z-wires as a number): " ++ show solvePart1 ++ "\n" ++
       "  Part 2: not solved (adder-structure puzzle, no example) - see README\n"
