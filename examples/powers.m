||prints a table of powers 2 to 5 of the numbers 1 to 20
||
|| Ported to Miracula: ^ becomes a local pow function, and the
|| justification/concat helpers from the Miranda stdenv are defined here.

concat2 [] = []
concat2 (a:x) = a ++ concat2 x

rep n c = [c | i <- [1..n]]
rjustify n s = rep (n - #s) ' ' ++ s
cjustify n s = rep lm ' ' ++ s ++ rep (n - #s - lm) ' '
               where lm = (n - #s) / 2

pow n 0 = 1
pow n k = n * pow n (k - 1)

format = rjustify 12

title = cjustify 60 "A TABLE OF POWERS" ++ "\n\n"
caption i = format ("N^" ++ show i)
captions = format "N" ++ concat2 (map caption [2..5]) ++ "\n"
line n = concat2 [format (show (pow n i)) | i <- [1..5]] ++ "\n"

main = title ++ captions ++ concat2 (map line [1..20])
