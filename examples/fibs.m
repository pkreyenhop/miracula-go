||This program tabulates the values of `fib i' a function for computing
||fibonacci numbers, in a list `fibs'.  Because the function is memoised
||(i.e. it looks its recursive calls up in the list) it runs in linear time.
||
|| Ported to Miracula: the n+2 pattern becomes a guard, ! indexing
|| becomes nth, and fib/fibs are mutually recursive so they live in one
|| where block (top-level definitions cannot forward-reference).

nth (a:x) n = a, if n == 0
            = nth x (n - 1), otherwise

fibs = f
       where
       f = map fib [0..]
       fib 0 = 0
       fib 1 = 1
       fib n = nth f (n - 1) + nth f (n - 2)

concat2 [] = []
concat2 (a:x) = a ++ concat2 x
layn xs = concat2 [show i ++ ") " ++ x ++ "\n" | (i, x) <- zip ([1..#xs], xs)]

main = layn (map show (take 20 fibs))
