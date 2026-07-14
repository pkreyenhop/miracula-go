||Finds pythagorean triangles (right triangles with integer sides).
||
|| Ported to Miracula: the original used a diagonalising comprehension
|| (// instead of |) over two infinite ranges, plus floating point
|| sqrt.  Miracula has neither, so we bound the hypotenuse and test
|| a*a + b*b == c*c directly with integers.

pyths = [(a, b, c) | c <- [1..40]; b <- [1..c]; a <- [1..b]; a*a + b*b == c*c]

lay2 [] = []
lay2 (a:x) = a ++ "\n" ++ lay2 x

main = lay2 (map show pyths)
