|| string == [char]

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
