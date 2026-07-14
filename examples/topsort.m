||Miranda programming example - topological sort
||topsort takes a list of pairs representing a partial order - where
||the presence of (u,v) in the list means that u precedes v - and
||returns a total ordering consistent with the information given.
||
|| Ported to Miracula: = becomes ==, mkset is defined here, the error
|| equation for cyclic data is dropped (hd of an empty list reports a
|| runtime error instead), and helpers precede their users.

mkset [] = []
mkset (a:x) = a : mkset [b | b <- x; b ~= a]

dom r = mkset [u | (u, v) <- r]
ran r = mkset [v | (u, v) <- r]
union x y = mkset (x ++ y)
carrier r = union (dom r) (ran r)

tsort c r = [], if c == []
          = a : tsort (c -- [a]) [(u, v) | (u, v) <- r; u ~= a], otherwise
            where
            a = hd m
            m = c -- ran r

topsort rel = tsort (carrier rel) rel

||dressing order: each pair means "put on the first before the second"
demo = [("socks", "shoes"), ("underpants", "trousers"),
        ("trousers", "shoes"), ("shirt", "jumper"),
        ("trousers", "jumper"), ("shirt", "trousers")]

main = show (topsort demo) ++ "\n"
