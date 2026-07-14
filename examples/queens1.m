||This finds one solution to the eight queens problem,  using  a
||different method from that of "queens.m".
||This time the backtracking is programmed explicitly.
||
|| Ported to Miracula: = becomes ==, board!i becomes nth, zip2 becomes
|| the tuple-argument zip, and until/and/abs/index are defined here.

nth (a:x) n = a, if n == 0
            = nth x (n - 1), otherwise

and2 [] = True
and2 (a:x) = a & and2 x

absv n = n, if n >= 0
       = 0 - n, otherwise

index xs = [0 .. #xs - 1]

until2 p f x = x, if p x
             = until2 p f (f x), otherwise

checks q board i = q == nth board i \/ absv (q - nth board i) == i + 1
safe (q:board) = and2 [~checks q board i | i <- index board]

alter (q:board) = q + 1 : board, if q < 8
                = alter board, otherwise ||backtrack

addqueen board = 1 : board
emptyboard = []
full board = #board == 8

extend board = until2 safe alter (addqueen board)
soln = until2 full extend emptyboard

concat2 [] = []
concat2 (a:x) = a ++ concat2 x

main = concat2 [c : show r ++ " " | (c, r) <- zip ("rnbqkbnr", soln)] ++ "\n"
