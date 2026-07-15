|| Advent of Code 2024 - Day 19: Linen Layout
|| https://adventofcode.com/2024/day/19
|| Given a set of towel patterns, each design must be built by concatenating
|| patterns. Part 1 counts how many designs are possible at all; part 2 sums
|| the number of distinct ways to build each design. Both use the same DP:
|| ways(suffix) = sum of ways(suffix without a matching leading pattern),
|| memoized on the suffix string (threaded through a map).
||
|| inputs/day19.txt is seeded with the official example (answers 6 / 16);
|| run fetch-inputs.sh to replace it with your personal puzzle input.

fstp (a, b) = a
sndp (a, b) = b

rawLines = [l | l <- lines (read "examples/aoc24/inputs/day19.txt"); l ~= ""]
patterns = split ", " (hd rawLines)
designs = tl rawLines

|| does `pat` prefix `s`? if so return (True, rest); else (False, s)
stripPrefix [] s = (True, s)
stripPrefix (p:ps) [] = (False, [])
stripPrefix (p:ps) (c:cs) = stripPrefix ps cs, if p == c
                          = (False, c : cs), otherwise

|| ways to build string s, memoized by s in the threaded map -> (count, map).
|| gather returns the list of per-pattern contributions (threading the memo);
|| summing a list sidesteps the interpreter's lazy-integer-accumulator bug.
ways s memo = (1, memo), if s == ""
            = (cached, memo), if cached ~= 0 - 1
            = (total, h_insert m2 s total), otherwise
              where
              cached = h_lookup_def memo s (0 - 1)
              gr = gather patterns memo
              total = sum (fstp gr)
              m2 = sndp gr
              gather [] mm = ([], mm)
              gather (p:ps) mm = gather ps mm, if ~ fstp sp
                               = (cnt : restCounts, m4), otherwise
                                 where
                                 sp = stripPrefix p s
                                 wr = ways (sndp sp) mm
                                 cnt = fstp wr
                                 tr = gather ps (sndp wr)
                                 restCounts = fstp tr
                                 m4 = sndp tr

|| a fresh memo per design (their suffixes rarely overlap); count via sum
waysOf d = fstp (ways d empty_map)
counts = [waysOf d | d <- designs]

solvePart1 = length [1 | c <- counts; c > 0]
solvePart2 = sum counts

main = "Advent of Code 2024 - Day 19 Results:\n" ++
       "  Part 1 (possible designs): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (total arrangements): " ++ show solvePart2 ++ "\n"
