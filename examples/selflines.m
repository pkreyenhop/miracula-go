||this produces an endless self describing scroll of lines, as follows
||	the 1st line is
||	"the 1st line is"
||	the 2nd line is
||	""the 1st line is""
||	etc...
||
|| Ported to Miracula: = becomes == in guards, ! indexing becomes nth,
|| div becomes /, and main takes a finite prefix of the infinite scroll.

nth (a:x) n = a, if n == 0
            = nth x (n - 1), otherwise

suffix n = "st", if n == 1
         = "nd", if n == 2
         = "rd", if n == 3
         = "th", if n == 0 \/ 3 <= n & n <= 9
         = "th", if (n mod 100) / 10 == 1
         = suffix (n mod 10), otherwise

ord n = show n ++ suffix n

selflines = mklines 1
            where
            mklines n = ("the " ++ ord n ++ " line is:") :
                        ("\"" ++ nth selflines (n - 1) ++ "\"") :
                        mklines (n + 1)

concat2 [] = []
concat2 (a:x) = a ++ concat2 x

main = concat2 [l ++ "\n" | l <- take 8 selflines]
