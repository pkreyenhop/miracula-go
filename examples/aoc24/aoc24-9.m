|| Advent of Code 2024 - Day 9: Disk Fragmenter
|| https://adventofcode.com/2024/day/9
|| The dense disk map alternates file-length and free-length digits. Part 1
|| compacts by moving single blocks from the end into the leftmost gaps; part 2
|| moves whole files (highest id first) into the leftmost gap that fits. The
|| answer is the checksum sum(position * fileId).
||
|| inputs/day9.txt is seeded with the official example (answers 1928 / 2858);
|| run fetch-inputs.sh to replace it with your personal puzzle input.

fstp (a, b) = a
sndp (a, b) = b
posFrom n c [] = 0 - 1
posFrom n c (x:xs) = if x == c then n else posFrom (n + 1) c xs
digitv c = posFrom 0 c "0123456789"

theline = hd (lines (read "examples/aoc24/inputs/day9.txt"))
digits = [digitv c | c <- theline]

|| segments as (id, start, len) for files and (start, len) for gaps.
|| walk the digits alternating file/gap, tracking running position and file id.
|| even index = a file (id fid), odd index = a gap.
segs even pos fid [] = ([], [])
segs even pos fid (d:ds) = (file : files, gaps), if even
                         = (files, gaps2), otherwise
                           where
                           rest = segs (~ even) (pos + d) (if even then fid + 1 else fid) ds
                           files = fstp rest
                           gaps = sndp rest
                           file = (fid, pos, d)
                           gaps2 = if d == 0 then gaps else (pos, d) : gaps

allsegs = segs True 0 0 digits
files = fstp allsegs
gaps = sndp allsegs

|| checksum of a file id occupying [start, start+len)
fchk fid start len = fid * (len * start + len * (len - 1) / 2)

|| ---- Part 1: single-block compaction via two-pointer over a block vector ----
replicate v 0 = []
replicate v n = v : replicate v (n - 1)

|| the flat block vector: file ids in disk order, -1 for each free block
blocks idx fid [] = []
blocks idx fid (d:ds) = if idx mod 2 == 0
                        then replicate fid d ++ blocks (idx + 1) (fid + 1) ds
                        else replicate (0 - 1) d ++ blocks (idx + 1) fid ds

blockVec = to_vec (blocks 0 0 digits)
nblocks = vec_len blockVec

solvePart1 = go 0 (nblocks - 1) 0
             where
             go l r acc = acc, if l > r
                        = go (l + 1) r (acc + l * vl), if vl >= 0
                        = go l (r - 1) acc, if vr < 0
                        = go (l + 1) (r - 1) (acc + l * vr), otherwise
                          where
                          vl = vec_get blockVec l
                          vr = vec_get blockVec r

|| ---- Part 2: whole-file moves, highest id first ----
|| place a file of length flen currently at fstart into the leftmost gap that
|| fits and lies left of the file; return (placedStart, updatedGaps).
place flen fstart [] = (fstart, [])
place flen fstart ((s, l):rest)
    = (fstart, (s, l):rest), if s >= fstart
    = (s, (s + flen, l - flen):rest), if l >= flen
    = (ps, (s, l):rest2), otherwise
      where
      pr = place flen fstart rest
      ps = fstp pr
      rest2 = sndp pr

|| fold files high id to low, threading the gap list, summing the checksum
moveFiles [] gs acc = acc
moveFiles ((fid, pos, len):fs) gs acc
    = moveFiles fs gs acc, if len == 0
    = moveFiles fs gs2 (acc + fchk fid ns len), otherwise
      where
      pr = place len pos gs
      ns = fstp pr
      gs2 = sndp pr

revfiles = reverse files
solvePart2 = moveFiles revfiles gaps 0

main = "Advent of Code 2024 - Day 9 Results:\n" ++
       "  Part 1 (block compaction checksum): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (whole-file compaction checksum): " ++ show solvePart2 ++ "\n"
