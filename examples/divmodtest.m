||This script defines tests for properties of division and mod, each
||checked over a small range of values including various combinations
||of signs.  Each test  should  yield  the  result  True, or there is
||something wrong with the arithmetic on your machine!
||
|| Ported to Miracula: there is no separate div (/ is floor division),
|| so the original test1 (a div b = entier (a/b)) has no analogue.
|| Multi-variable generators a,b <- ... become two generators, = becomes
|| ==, and the chained comparisons are split with &.

and2 [] = True
and2 (a:x) = a & and2 x

test2 = and2 [b * (a / b) + a mod b == a | a <- [(-15)..15]; b <- [(-15)..15]; b ~= 0]

test3 = and2 [ok a b | a <- [(-15)..15]; b <- [(-15)..15]; b ~= 0]
        where
        ok a b = 0 <= a mod b & a mod b < b, if b > 0
               = b < a mod b & a mod b <= 0, if b < 0

main = "test2 = " ++ show test2 ++ "\ntest3 = " ++ show test3 ++ "\n"
