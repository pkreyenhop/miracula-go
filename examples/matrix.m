||very simple matrix package (DT), matrices as lists of rows
||
|| Ported to Miracula: integer matrices only, so det and adjoint are
|| exact but inv (which divides by the determinant) is omitted.
|| ! indexing becomes nth, (-1)^(i+j) becomes a parity sign, and the
|| stdenv helpers (map2, sections) are written out at top level.

nth (a:x) n = a, if n == 0
            = nth x (n - 1), otherwise

index xs = [0 .. #xs - 1]

map2 f (a:x) (b:y) = f a b : map2 f x y
map2 f x y = []

plus2 p q = p + q
minus2 p q = p - q
mul2 p q = p * q

transpose xss = [], if live == []
              = map hd live : transpose (map tl live), otherwise
                where live = [xs | xs <- xss; xs ~= []]

idmat n = [[delta i j | j <- [1..n]] | i <- [1..n]]
          where
          delta i j = 1, if i == j
                    = 0, otherwise

vadd x y = map2 plus2 x y
vsub x y = map2 minus2 x y
matadd x y = map2 vadd x y
matsub x y = map2 vsub x y

inner x y = sum (map2 mul2 x y)
outer f x y = [[f p q | q <- y] | p <- x]
matmult x y = outer inner x (transpose y)

scalmult n x = map (map times) x
               where times p = n * p

mkrow x = [x]
mkcol x = map (: []) x

omit i x = take i x ++ drop (i + 1) x
minor i j xs = [omit j x | x <- omit i xs]

sign i j = 1, if (i + j) mod 2 == 0
         = 0 - 1, otherwise

det xs = d xs
         where
         d ys = hd (hd ys), if #ys == 1
              = sum [nth (nth ys 0) i * cof 0 i ys | i <- index ys], otherwise
         cof i j ys = sign i j * d (minor i j ys)

cofactor i j xs = sign i j * det (minor i j xs)

adjoint xs = transpose [[cofactor i j xs | j <- index xs] | i <- index xs]

ma = [[1, 2], [3, 4]]
mb = [[1, 1, 1], [1, 2, 3], [2, 4, 8]]

main = "det ma       = " ++ show (det ma) ++ "\n" ++
       "det mb       = " ++ show (det mb) ++ "\n" ++
       "adjoint ma   = " ++ show (adjoint ma) ++ "\n" ++
       "ma x adj ma  = " ++ show (matmult ma (adjoint ma)) ++ "\n" ++
       "ma + ma      = " ++ show (matadd ma ma) ++ "\n" ++
       "ma x id      = " ++ show (matmult ma (idmat 2)) ++ "\n"
