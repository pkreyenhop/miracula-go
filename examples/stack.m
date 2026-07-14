||This script defines stack, based on lists.  In Miranda this is an
||abstract data type; note the show function for stacks, causing them
||to print in a sensible way.
||
|| Ported to Miracula: no abstype, so these are ordinary functions over
|| lists (the representation is not hidden), and = becomes ==.

empty = []
push a x = a : x
pop (a:x) = x
top (a:x) = a
isempty x = (x == [])
showstack f [] = "empty"
showstack f (a:x) = "(push " ++ f a ++ " " ++ showstack f x ++ ")"

teststack = push 1 (push 2 (push 3 empty))

main = showstack show teststack ++ "\n" ++
       "top = " ++ show (top teststack) ++ "\n" ++
       "after pop: " ++ showstack show (pop teststack) ++ "\n"
