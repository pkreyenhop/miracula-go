||Programming example - generating the digits of `e', as an infinite
||string.  e = 1 + 1/1! + 1/2! + 1/3! + ...  In a base system where the
||weight of the i'th fractional digit is 1/i!, e is just 2.111111...,
||so we need only convert that fraction to decimal.  See the original
||literate script for the full derivation, including why norm must be
||careful about how far a carry can propagate (max carry is 9).
||
|| Ported to Miracula: the literate > markers are gone, decode/code
|| become indexing into a digit string, map (10*) becomes a
|| comprehension, and the (e':x') cons binding in a where clause is
|| replaced by hd/tl projections.  main takes 60 digits.

nth (a:x) n = a, if n == 0
            = nth x (n - 1), otherwise

mkdigit n = nth "0123456789" n, if n < 10

norm c (d:e:x) = d + e / c : e2 mod c : x2, if e mod c + 9 < c
               = d + e2 / c : e2 mod c : x2, otherwise
                 where
                 r = norm (c + 1) (e : x)
                 e2 = hd r
                 x2 = tl r

convert x = mkdigit (hd x2) : convert (tl x2)
            where x2 = norm 2 (0 : [10 * d | d <- x])

edigits = "2." ++ convert (repeat 1)

main = take 60 edigits ++ "\n"
