||Text formatting program (DT)
||Reformats text to a specified width, with line-fill.
||In this program we move between three representations of text -
||as a flat character list, as a list of lines, and as a list of
||lists of words.  The words/unwords definitions are due to Richard
||Bird, see Bird and Wadler (1988), page 91.
||
|| Ported to Miracula: = becomes ==, chained relations are split,
|| sections like (~=[]) and (#) become named helpers, the (x1,rest)
|| tuple binding in a where clause becomes fst/snd projections, the
|| step range in mkspaces becomes reverse [1..n], and Bird's ragged
|| transpose trick in fill_line becomes an explicit interleave.
|| main formats an embedded sample instead of reading a file.

concat2 [] = []
concat2 (a:x) = a ++ concat2 x

rep n c = [c | i <- [1..n]]

foldr2 f z [] = z
foldr2 f z (a:x) = f a (foldr2 f z x)

foldr1x f (a:x) = a, if x == []
                = f a (foldr1x f x), otherwise

init2 (a:x) = [], if x == []
            = a : init2 x, otherwise

last2 (a:x) = a, if x == []
            = last2 x, otherwise

nonempty w = w ~= []

breakon c a x = [] : x, if a == c
              = (a : hd x) : tl x, otherwise

words2 s = filter nonempty (foldr2 (breakon ' ') [[]] s)

insertsep a x y = x ++ [a] ++ y
unwords2 ws = foldr1x (insertsep ' ') ws

lay2 [] = []
lay2 (a:x) = a ++ "\n" ++ lay2 x

||return s spaces broken into n groups; the extra spaces go sometimes
||to the left and sometimes to the right, to keep the text balanced
mkspaces n s = map f [1..n], if n mod 2 == 0
             = map f (reverse [1..n]), otherwise
               where
               f i = rep (s / n + 1) ' ', if i <= s mod n
                   = rep (s / n) ' ', otherwise

weave (w:ws) (s:ss) = w : s : weave ws ss
weave ws [] = ws
weave [] ss = ss

||make words into a line of length n exactly, by inserting spaces
fill_line n ws = concat2 (weave ws (mkspaces (w - 1) (n - sw)))
                 where
                 w = #ws
                 sw = sum [#x | x <- ws]

fst2 (a, b) = a
snd2 (a, b) = b

grab n y (w:x) = grab n (w : y) x, if sum [#u | u <- y] + #y + #w <= n
               = (reverse y, w : x), otherwise
grab n y [] = (reverse y, [])

||break a paragraph into lines with as many words as fit in width n
partition n [] = []
partition n x = fst2 g : partition n (snd2 g)
                where g = grab n [] x

justify n para = map (fill_line n) (init2 para) ++ [unwords2 (last2 para)]

||reformat one paragraph to width n; an empty one is a blank line
reformat n [] = "\n"
reformat n x = lay2 (justify n (partition n x))

||join adjacent non-blank lines into paragraphs
paras (a:b:x) = paras ((a ++ b) : x), if a ~= [] & b ~= []
              = a : paras (b : x), otherwise
paras (a:[]) = [a]
paras [] = []

just n t = concat2 (map (reformat n) (paras (map words2 (lines t))))

sample = "The quick brown fox jumps over the lazy dog. Pack my box with five dozen liquor jugs. How vexingly quick daft zebras jump!\n\nSphinx of black quartz, judge my vow. The five boxing wizards jump quickly over the truly lazy dwarf standing in the corner.\n"

main = just 40 sample
