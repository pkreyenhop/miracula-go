|| string == [char]

|| the circle constant, as a real (Miranda's `pi`)
pi = 3.141592653589793

|| strict in the accumulator (via seq) so long folds run in constant space
foldl f z []     = z
foldl f z (x:xs) = seq z2 (foldl f z2 xs)
                   where
                   z2 = f z x

converse f a b = f b a

sum = foldl (+) 0

product = foldl mul 1
          where mul a b = a * b

map f x = [f a | a<-x]

filter p xs = [x | x <- xs; p x]

foldr f z []     = z
foldr f z (x:xs) = f x (foldr f z xs)

take 0 xs     = []
take n []     = []
take n (x:xs) = x : take (n-1) xs

drop 0 xs     = xs
drop n []     = []
drop n (x:xs) = drop (n-1) xs

takewhile p []     = []
takewhile p (x:xs) = if p x then x : takewhile p xs else []

iterate f x = x : iterate f (f x)

repeat x = x : repeat x

zip ([], []) = []
zip (x:xs, y:ys) = (x, y) : zip (xs, ys)

|| ==========================================================================
|| Functions adapted from the Miranda standard environment (stdenv-miranda.m).
|| Only the parts expressible in Miracula are given here; definitions must
|| precede their users (top-level scoping is sequential). Parts that cannot be
|| expressed are listed, with the reason, in the commented block at the end.
|| ==========================================================================

|| --- combinators -----------------------------------------------------------
id x = x
const x y = x
fst (a, b) = a
snd (a, b) = b

|| --- numeric (integer only; Miranda's float behaviour is not available) ----
neg x = 0 - x
abs x = 0 - x, if x < 0
      = x, otherwise
subtract x y = y - x         || (converse) infix minus: subtract 3 subtracts 3

|| --- logical folds over lists ----------------------------------------------
|| (`&` / `\/` have no section form, so a small local operator is used)
and bs = foldr conj True bs
         where conj a b = a & b
or bs = foldr disj False bs
        where disj a b = a \/ b

|| --- list building ---------------------------------------------------------
concat xss = foldr app [] xss
             where app a b = a ++ b
postfix a x = x ++ [a]        || dual of the prefix operator ':'

|| --- folds over non-empty lists (the Miranda `error []` case is omitted, so
||     applying these to [] raises "Pattern matching exhausted") -------------
foldl1 op (a:x) = foldl op a x
foldr1 op (a:[]) = a
foldr1 op (a:b:x) = op a (foldr1 op (b:x))

|| --- ordering (polymorphic and structural: <, <=, > work on any one type) --
max2 a b = a, if a >= b
         = b, otherwise
min2 a b = b, if a > b
         = a, otherwise
max xs = foldl1 max2 xs       || largest element under the built-in '>'
min xs = foldl1 min2 xs       || least element under the built-in '<'

|| --- character predicates --------------------------------------------------
digit x = '0' <= x & x <= '9'
letter c = ('a' <= c & c <= 'z') \/ ('A' <= c & c <= 'Z')

|| --- dropwhile (takewhile's dual is already native above) -------------------
dropwhile f [] = []
dropwhile f (a:x) = dropwhile f x, if f a
                  = a : x, otherwise

|| --- merge sort on the built-in ordering (n log n) -------------------------
merge [] y = y
merge (a:x) [] = a : x
merge (a:x) (b:y) = a : merge x (b:y), if a <= b
                  = b : merge (a:x) y, otherwise
sort x = x, if n <= 1
       = merge (sort (take n2 x)) (sort (drop n2 x)), otherwise
         where
         n = #x
         n2 = n / 2

|| --- mkset: remove duplicates (quadratic; works on infinite lists) ---------
mkset [] = []
mkset (a:x) = a : filter neqa (mkset x)
              where neqa e = e ~= a

|| --- replication and field justification -----------------------------------
rep n x = take n (repeat x)   || rep 6 'o' = "oooooo"
spaces n = rep n ' '
ljustify n s = s ++ spaces (n - #s)
rjustify n s = spaces (n - #s) ++ s
cjustify n s = spaces lmargin ++ s ++ spaces rmargin
               where
               margin = n - #s
               lmargin = margin / 2
               rmargin = margin - lmargin

|| --- indexing and end-of-list helpers --------------------------------------
index x = f 0 x               || legal subscripts of x, in ascending order
          where
          f n [] = []
          f n (a:y) = n : f (n + 1) y
init (a:x) = [], if x == []   || the list without its last element
           = a : init x, otherwise
last (a:x) = a, if x == []    || the last element
           = last x, otherwise

|| --- line-oriented formatting (inverse of the native `lines`) --------------
lay [] = []
lay (a:x) = a ++ "\n" ++ lay x
layn x = f 1 x                || like lay, but numbers the lines
         where
         f n [] = []
         f n (a:y) = rjustify 4 (show n) ++ ") " ++ a ++ "\n" ++ f (n + 1) y

|| --- limit: first value equal to its successor (convergence testing) -------
limit (a:b:x) = a, if a == b
              = limit (b:x), otherwise

|| --- until and scan --------------------------------------------------------
until f g x = x, if f x       || apply g until f holds: until (>1000) (2*) 1
            = until f g (g x), otherwise
scan op r [] = [r]            || running foldl over every initial segment
scan op r (a:x) = r : scan op (op r a) x

|| --- two-list map and transpose -------------------------------------------
map2 f x y = [f a b | (a, b) <- zip (x, y)]
|| zipWith is the curried Bird-and-Wadler form of map2; zip2 is a curried zip
zipWith f (x:xs) (y:ys) = f x y : zipWith f xs ys
zipWith f xs ys = []
zip2 xs ys = zipWith pair xs ys
             where pair a b = (a, b)
transpose x = [], if xp == []
            = map hd xp : transpose (map tl xp), otherwise
              where
              nonempty e = e ~= []
              xp = takewhile nonempty x

|| --- wider zips (zip itself takes a tuple; these are curried like Miranda) --
zip3 (a:x) (b:y) (c:z) = (a, b, c) : zip3 x y z
zip3 x y z = []
zip4 (a:w) (b:x) (c:y) (d:z) = (a, b, c, d) : zip4 w x y z
zip4 w x y z = []
zip5 (a:v) (b:w) (c:x) (d:y) (e:z) = (a, b, c, d, e) : zip5 v w x y z
zip5 v w x y z = []
zip6 (a:u) (b:v) (c:w) (d:x) (e:y) (f:z) = (a, b, c, d, e, f) : zip6 u v w x y z
zip6 u v w x y z = []

|| ==========================================================================
|| Not expressible in Miracula (kept here as a record of what was left out):
||
||   Already provided natively or above, so not re-defined here:
||     hd, tl, reverse, length (#), seq, map, filter, foldl, foldr, take,
||     drop, takewhile, iterate, repeat, zip, sum, product, converse, lines,
||     read, numval, sort_by/sort_ints (native sorts).
||
||   Floating point / transcendental — `num` is a 64-bit integer, no floats:
||     abs (float form), arctan, cos, sin, exp, log, log10, sqrt, pi, e,
||     entier, integer, hugenum, tinynum.
||
||   No error / undefined values (so the folds above are simply partial):
||     error, undef, force.
||
||   No char<->code conversion built in:  code, decode.
||
||   No user type declarations / algebraic types:
||     bool, char, num, sys_message.
||
||   Number / radix formatting is internal to `show`:
||     shownum, showhex, showoct, showfloat, showscaled.
||
||   UNIX interface not available (except the native `read`):
||     filemode, filestat, getenv, system, readb.
||
||   member — Miranda's list membership `[*]->*->bool`; the name `member` is
||     the native set-membership builtin and cannot be redefined at top level.
||     For a list version write, in your own script:
||       elem a [] = False
||       elem a (x:xs) = x == a \/ elem a xs
|| ==========================================================================
