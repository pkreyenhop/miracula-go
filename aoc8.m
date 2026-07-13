|| Advent of Code 2025 - Day 8: Euclidean Circuit Clustering (Pure Miracula)

seq x y = ifzero x then y else y
seq_tuple (a, b, c, d) y = seq a (seq b (seq c (seq d y)))

fst (a, b) = a
snd (a, b) = b

fst3 (a, b, c) = a
snd3 (a, b, c) = b
thd3 (a, b, c) = c

fst4 (a, b, c, d) = a
snd4 (a, b, c, d) = b
thd4 (a, b, c, d) = c
fth4 (a, b, c, d) = d

split_by5 c [] = [[]]
split_by5 c (x:xs) = if x == c then [] : split_by5 c xs else (x : hd rest) : tl rest
                    where
                    rest = split_by5 c xs

not_empty5 l = l ~= ""

distSq8 (x1, y1, z1) (x2, y2, z2) = dx*dx + dy*dy + dz*dz
  where
  dx = x1 - x2
  dy = y1 - y2
  dz = z1 - z2

insert_edge8 j d (m1_idx, m1_d, m2_idx, m2_d) =
  if d < m1_d then (j, d, m1_idx, m1_d)
  else (if d < m2_d then (m1_idx, m1_d, j, d)
        else (m1_idx, m1_d, m2_idx, m2_d))

scan_right px py pz [] acc = acc
scan_right px py pz (p : rest) acc =
  if dx * dx >= m2_d then acc
  else seq_tuple next (scan_right px py pz rest next)
  where
  j = fst p
  coords = snd p
  jx = fst3 coords
  jy = snd3 coords
  jz = thd3 coords
  dx = jx - px
  m2_d = fth4 acc
  next = insert_edge8 j (distSq8 (px,py,pz) (jx,jy,jz)) acc

scan_left px py pz [] acc = acc
scan_left px py pz (p : rest) acc =
  if dx * dx >= m2_d then acc
  else seq_tuple next (scan_left px py pz rest next)
  where
  j = fst p
  coords = snd p
  jx = fst3 coords
  jy = snd3 coords
  jz = thd3 coords
  dx = px - jx
  m2_d = fth4 acc
  next = insert_edge8 j (distSq8 (px,py,pz) (jx,jy,jz)) acc

find_all_nearest_sweep lefts [] = []
find_all_nearest_sweep lefts (p : rights) =
  (min i j1, max i j1, d1) : (min i j2, max i j2, d2) : find_all_nearest_sweep (p : lefts) rights
  where
  i = fst p
  coords = snd p
  px = fst3 coords
  py = snd3 coords
  pz = thd3 coords
  
  j_l = if lefts == [] then 0-1 else fst (hd lefts)
  d_l = if lefts == [] then 2000000000 else distSq8 (px,py,pz) (snd (hd lefts))
  
  j_r = if rights == [] then 0-1 else fst (hd rights)
  d_r = if rights == [] then 2000000000 else distSq8 (px,py,pz) (snd (hd rights))
  
  acc0 = if d_l < d_r then (j_l, d_l, j_r, d_r) else (j_r, d_r, j_l, d_l)
  
  acc1 = scan_left px py pz lefts acc0
  res = scan_right px py pz rights acc1
  j1 = fst4 res
  d1 = snd4 res
  j2 = thd4 res
  d2 = fth4 res
  min a b = if a < b then a else b
  max a b = if a > b then a else b

qsort_pts [] = []
qsort_pts (p:ps) = qsort_pts lt ++ (p : eq) ++ qsort_pts gt
  where
  res = partition (fst3 (snd p)) ps ([], [], [])
  lt = fst3 res
  eq = snd3 res
  gt = thd3 res
  
  partition pivot [] (lt, eq, gt) = (lt, eq, gt)
  partition pivot (y:ys) (lt, eq, gt) =
    if d < pivot then partition pivot ys (y:lt, eq, gt)
    else (if d == pivot then partition pivot ys (lt, y:eq, gt)
          else partition pivot ys (lt, eq, y:gt))
    where
    d = fst3 (snd y)

qsort8 [] = []
qsort8 (x:xs) = qsort8 lt ++ (x : eq) ++ qsort8 gt
  where
  res = partition (thd3 x) xs ([], [], [])
  lt = fst3 res
  eq = snd3 res
  gt = thd3 res
  
  partition p [] (lt, eq, gt) = (lt, eq, gt)
  partition p (y:ys) (lt, eq, gt) =
    if d < p then partition p ys (y:lt, eq, gt)
    else (if d == p then partition p ys (lt, y:eq, gt)
          else partition p ys (lt, eq, y:gt))
    where
    d = thd3 y

dedup8 [] = []
dedup8 (x:xs) = if xs == [] then [x]
                else (if (fst3 x == fst3 y) & (snd3 x == snd3 y) then dedup8 (x : tl xs)
                      else x : dedup8 xs)
                where
                y = hd xs

get_parent 0 (x:xs) = x
get_parent k (x:xs) = get_parent (k-1) xs

update_parent 0 val (x:xs) = val : xs
update_parent k val (x:xs) = x : update_parent (k-1) val xs

find_root parent x =
  if p == x then (x, parent)
  else (root, update_parent x root root_parent)
  where
  p = get_parent x parent
  res = find_root parent p
  root = fst res
  root_parent = snd res

kruskal8 [] parent count = parent
kruskal8 (e:es) parent count =
  if count >= 1000 then parent
  else (if ru == rv then kruskal8 es parent_v count
        else kruskal8 es (update_parent ru rv parent_v) (count + 1))
  where
  u = fst3 e
  v = snd3 e
  res_u = find_root parent u
  ru = fst res_u
  parent_u = snd res_u
  res_v = find_root parent_u v
  rv = fst res_v
  parent_v = snd res_v

group_count [] = []
group_count (x:xs) = count_more 1 x xs
  where
  count_more k val [] = [k]
  count_more k val (y:ys) = if y == val then count_more (k+1) val ys
                            else k : count_more 1 y ys

qsort_desc8 [] = []
qsort_desc8 (x:xs) = qsort_desc8 gt ++ (x : eq) ++ qsort_desc8 lt
  where
  res = partition x xs ([], [], [])
  lt = fst3 res
  eq = snd3 res
  gt = thd3 res
  
  partition pivot [] (lt, eq, gt) = (lt, eq, gt)
  partition pivot (y:ys) (lt, eq, gt) =
    if y < pivot then partition pivot ys (y:lt, eq, gt)
    else (if y == pivot then partition pivot ys (lt, y:eq, gt)
          else partition pivot ys (lt, eq, y:gt))

qsort_asc [] = []
qsort_asc (x:xs) = qsort_asc lt ++ (x : eq) ++ qsort_asc gt
  where
  res = partition x xs ([], [], [])
  lt = fst3 res
  eq = snd3 res
  gt = thd3 res
  
  partition pivot [] (lt, eq, gt) = (lt, eq, gt)
  partition pivot (y:ys) (lt, eq, gt) =
    if y < pivot then partition pivot ys (y:lt, eq, gt)
    else (if y == pivot then partition pivot ys (lt, y:eq, gt)
          else partition pivot ys (lt, eq, y:gt))

parse_point8 line = (numval (hd parts), numval (hd (tl parts)), numval (hd (tl (tl parts))))
  where
  parts = split_by5 ',' line

solvePart1 input = s1 * s2 * s3
  where
  pts = map parse_point8 (filter not_empty5 (split_by5 '\n' input))
  indexed_pts = zip ([0 .. length pts - 1], pts)
  sorted_pts = qsort_pts indexed_pts
  edges = dedup8 (qsort8 [ e | e <- find_all_nearest_sweep [] sorted_pts; fst3 e >= 0 ])
  
  initial_parent = [ i | i <- [0 .. length pts - 1] ]
  final_parent = kruskal8 edges initial_parent 0
  
  find_all_roots [] p = ([], p)
  find_all_roots (x:xs) p = (r : rest_roots, final_p)
    where
    res_x = find_root p x
    r = fst res_x
    p_x = snd res_x
    res_xs = find_all_roots xs p_x
    rest_roots = fst res_xs
    final_p = snd res_xs
    
  roots_res = find_all_roots [0 .. length pts - 1] final_parent
  roots = fst roots_res
  sorted_roots = qsort_asc roots
  sizes = group_count sorted_roots
  sorted_sizes = qsort_desc8 sizes
  
  s1 = hd sorted_sizes
  s2 = hd (tl sorted_sizes)
  s3 = hd (tl (tl sorted_sizes))

main = "Advent of Code 2025 - Day 8 Results:\n" ++
       "  Part 1 (Circuit size product): " ++ show p1Result ++ "\n"
       where
       input = read "inputs/day8-example.txt"
       p1Result = solvePart1 input
