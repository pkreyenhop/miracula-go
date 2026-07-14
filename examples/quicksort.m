||this is a functional version of quicksort, to see it work, say:
||	qsort testdata
||
|| Ported to Miracula: the step ranges [0,2..2n] and [2n-1,2n-3..1] in
|| testdata are written as comprehensions, and transpose/concat are
|| defined here (they are not in the Miracula standard environment).

qsort [] = []
qsort (a:x) = qsort [b | b <- x; b <= a] ++ [a] ++ qsort [b | b <- x; b > a]

transpose xss = [], if live == []
              = map hd live : transpose (map tl live), otherwise
                where live = [xs | xs <- xss; xs ~= []]

concat2 [] = []
concat2 (a:x) = a ++ concat2 x

f n = concat2 (transpose [[2 * i | i <- [0..n]], [2 * (n - i) - 1 | i <- [0..n - 1]]])
testdata = f 10

main = show (qsort testdata) ++ "\n"
