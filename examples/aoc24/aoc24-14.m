|| Advent of Code 2024 - Day 14: Restroom Redoubt
|| https://adventofcode.com/2024/day/14
|| Robots wrap around a WIDTH x HEIGHT grid. Part 1: after 100 seconds, count
|| robots in each quadrant (ignoring the centre row/column) and multiply.
|| Part 2: robots briefly form a Christmas tree; that frame has them unusually
|| clustered, so we report the second (in [0, WIDTH*HEIGHT)) with the smallest
|| quadrant "safety factor".
||
|| IMPORTANT: the seeded example uses an 11 x 7 grid (part 1 answer 12). For
|| your real puzzle input set WIDTH = 101 and HEIGHT = 103 below. The tree in
|| part 2 only appears on the real 101 x 103 grid.
||
|| inputs/day14.txt is seeded with the official example; run fetch-inputs.sh
|| to replace it with your personal puzzle input (and switch WIDTH/HEIGHT).

width = 11
height = 7

posmod a m = ((a mod m) + m) mod m

|| each line -> (px, py, vx, vy)
toRobot (px:py:vx:vy:rest) = (px, py, vx, vy)
robots = [toRobot (parse_ints l) | l <- lines (read "examples/aoc24/inputs/day14.txt"); l ~= ""]

|| position of a robot after t seconds
posAt t (px, py, vx, vy) = (posmod (px + t * vx) width, posmod (py + t * vy) height)

|| safety factor: product of the four quadrant counts at time t
safety t = q 0 0 * q 1 0 * q 0 1 * q 1 1
           where
           cx = width / 2
           cy = height / 2
           ps = [posAt t rb | rb <- robots]
           q sx sy = length [1 | (x, y) <- ps;
                                 (if sx == 0 then x < cx else x > cx);
                                 (if sy == 0 then y < cy else y > cy)]

solvePart1 = safety 100

|| the frame with the lowest safety factor (the tree, on the real grid)
best = hd (sort_by cmp [(safety t, t) | t <- [0 .. width * height - 1]])
       where cmp p qq = fst p - fst qq
solvePart2 = snd best

main = "Advent of Code 2024 - Day 14 Results:\n" ++
       "  Part 1 (safety factor after 100s): " ++ show solvePart1 ++ "\n" ++
       "  Part 2 (min-safety second, tree on real grid): " ++ show solvePart2 ++ "\n"
