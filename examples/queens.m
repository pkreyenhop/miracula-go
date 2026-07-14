||this generates all solutions to the 8 queens problem -- and prints
||the solutions one per line - all 92 of them.
||
|| Ported to Miracula: the n+1 pattern becomes a guard, = becomes ==,
|| b!i becomes nth, and the stdenv helpers (and, abs, index, layn) are
|| defined here.

nth (a:x) n = a, if n == 0
            = nth x (n - 1), otherwise

and2 [] = True
and2 (a:x) = a & and2 x

absv n = n, if n >= 0
       = 0 - n, otherwise

index xs = [0 .. #xs - 1]

checks q b i = q == nth b i \/ absv (q - nth b i) == i + 1
safe q b = and2 [~checks q b i | i <- index b]

queens n = [[]], if n == 0
         = [q:b | b <- queens (n - 1); q <- [1..8]; safe q b], otherwise

concat2 [] = []
concat2 (a:x) = a ++ concat2 x
layn xs = concat2 [show i ++ ") " ++ x ++ "\n" | (i, x) <- zip ([1..#xs], xs)]

main = layn (map show (queens 8))
