||rational numbers, kept in their lowest terms with positive
||denominator; (0,1) is the unique representation of zero.
||
|| Ported to Miracula: in Miranda this is an abstract data type with
|| $infix operators; here rationals are plain (num,num) pairs with
|| prefix functions, unary minus is written 0-, and the error/integer
|| checks are dropped (all Miracula numbers are integers).

absv n = n, if n >= 0
       = 0 - n, otherwise

ratio p q = ratio (0 - p) (0 - q), if q < 0
          = (0, 1), if p == 0
          = (p / h, q / h), otherwise
            where
            h = hcf (absv p) q
            hcf a b = hcf b a, if a > b
                    = b, if a == 0
                    = hcf (b mod a) a, otherwise

mkrat n = ratio n 1

rplus (a, b) (c, d) = ratio (a*d + c*b) (b*d)
rminus (a, b) (c, d) = ratio (a*d - c*b) (b*d)
rtimes (a, b) (c, d) = ratio (a*c) (b*d)
rdiv (a, b) (c, d) = ratio (a*d) (b*c)

rpow n x = (1, 1), if n == 0
         = rtimes t t, if n mod 2 == 0
         = rtimes x (rpow (n - 1) x), otherwise
           where
           t = rpow (n / 2) x

numerator (a, b) = a
denominator (a, b) = b

showrational (a, b) = show a, if b == 1
                    = show a ++ "/" ++ show b, otherwise

half = ratio 1 2
third = ratio 1 3
twothirds = ratio 2 3

main = "1/2 + 1/3 = " ++ showrational (rplus half third) ++ "\n" ++
       "1/2 - 1/3 = " ++ showrational (rminus half third) ++ "\n" ++
       "(2/3)^5   = " ++ showrational (rpow 5 twothirds) ++ "\n" ++
       "(1/2)/(2/3) = " ++ showrational (rdiv half twothirds) ++ "\n"
