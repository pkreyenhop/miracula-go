||definition of finite sets, represented as ordered lists without
||duplicates.
||
|| Ported to Miracula: no abstype (these are ordinary functions over
|| sorted lists), = becomes ==, a generic insertion sort replaces the
|| stdenv sort, mem keeps its name (the native builtin over Miracula's
|| own set type is called member), and the error equations are dropped.

isort [] = []
isort (a:x) = ins a (isort x)
              where
              ins b [] = [b]
              ins b (c:y) = b : c : y, if b <= c
                          = c : ins b y, otherwise

uniq (a:x) = a : uniq [b | b <- x; b ~= a], if x ~= []
           = [a], otherwise
uniq [] = []

merge2 (a:x) (b:y) = a : merge2 x (b:y), if a <= b
                   = b : merge2 (a:x) y, otherwise
merge2 [] y = y
merge2 x [] = x

foldr2 f z [] = z
foldr2 f z (a:x) = f a (foldr2 f z x)

foldl1 f (a:x) = a, if x == []
               = foldl1 f (f a (hd x) : tl x), otherwise

makeset x = uniq (isort x)
enum x = x
empty = []
mem (a:x) b = a == b \/ (a < b & mem x b)
mem [] b = False
setdiff (a:x) (b:y) = a : setdiff x (b:y), if a < b
                    = setdiff (a:x) y, if a > b
                    = setdiff x y, otherwise
setdiff x y = x
includes x y = (setdiff y x == [])
pincludes x y = x ~= y & (setdiff y x == [])
union2 x y = uniq (merge2 x y)
union x = foldr2 union2 empty x
intersect2 (a:x) (b:y) = intersect2 x (b:y), if a < b
                       = intersect2 (a:x) y, if a > b
                       = a : intersect2 x y, otherwise
intersect2 x y = []
intersect = foldl1 intersect2
add1 a (b:x) = a : b : x, if a < b
             = b : x, if a == b
             = b : add1 a x, otherwise
add1 a [] = [a]
sub1 a (b:x) = b : x, if a < b
             = x, if a == b
             = b : sub1 a x, otherwise
sub1 a [] = []
pick (a:x) = a
rest (a:x) = x

concat2 [] = []
concat2 (a:x) = a ++ concat2 x

showset f [] = "{}"
showset f (a:x) = "{" ++ f a ++ concat2 (map g x) ++ "}"
                  where
                  g b = ',' : f b

s1 = makeset [3, 1, 4, 1, 5, 9, 2, 6]
s2 = makeset [2, 7, 1, 8, 2, 8]

main = "s1            = " ++ showset show s1 ++ "\n" ++
       "s2            = " ++ showset show s2 ++ "\n" ++
       "union         = " ++ showset show (union [s1, s2]) ++ "\n" ++
       "intersect     = " ++ showset show (intersect [s1, s2]) ++ "\n" ++
       "s1 -- s2      = " ++ showset show (setdiff s1 s2) ++ "\n" ++
       "mem s1 4      = " ++ show (mem s1 4) ++ "\n" ++
       "add1 7        = " ++ showset show (add1 7 s1) ++ "\n"
