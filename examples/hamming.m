||this is a problem described by Dijkstra in his book, A Discipline of
||Programming, and attributed by him to Dr Hamming, of Bell Labs.
||Print in ascending order all numbers of the form
|| 2**a.3**b.5**c        a,b,c all >=0
||the solution here is based on a method using communicating processes.
||
|| Ported to Miracula: = becomes == in the guards, foldr1 is defined
|| here, and main takes a finite prefix of the infinite list.

ham = 1 : foldr1 merge [mult 2 ham, mult 3 ham, mult 5 ham]
      where
      mult n x = [n * a | a <- x]
      foldr1 f (a:x) = a, if x == []
                     = f a (foldr1 f x), otherwise
      merge (a:x) (b:y) = a : merge x y,     if a == b
                        = a : merge x (b:y), if a < b
                        = b : merge (a:x) y, if a > b

main = show (take 30 ham) ++ "\n"
