||This script generates a solution to the well known `Towers of Hanoi'
||problem.
||
|| Ported to Miracula: n+k pattern replaced by a guard; 12 discs
|| trimmed to 5 so the output fits a screen.

move a b = "move the top disc from " ++ a ++ " to " ++ b ++ "\n"

hanoi n a b c = [], if n == 0
              = hanoi (n-1) a c b
                ++ move a b ++
                hanoi (n-1) c b a, otherwise

title = "SOLUTION TO TOWERS OF HANOI WITH 5 DISCS\n\n"
main = title ++ hanoi 5 "A" "B" "C"
