|| Advent of Code 2025 - Day 8: Euclidean Circuit Clustering (Pure Miracula)

seq x y = ifzero x then y else y

fst (a, b) = a
snd (a, b) = b

fst3 (a, b, c) = a
snd3 (a, b, c) = b
thd3 (a, b, c) = c

fst4 (a, b, c, d) = a
snd4 (a, b, c, d) = b
thd4 (a, b, c, d) = c
fth4 (a, b, c, d) = d

distSq8 (x1, y1, z1) (x2, y2, z2) = dx*dx + dy*dy + dz*dz
  where
  dx = x1 - x2
  dy = y1 - y2
  dz = z1 - z2

scan_right px py pz [] m1_idx m1_d m2_idx m2_d = (m1_idx, m1_d, m2_idx, m2_d)
scan_right px py pz ((j, coords) : rest) m1_idx m1_d m2_idx m2_d =
  if dx * dx >= m2_d then (m1_idx, m1_d, m2_idx, m2_d)
  else seq next_m1_idx (seq next_m1_d (seq next_m2_idx (seq next_m2_d
       (scan_right px py pz rest next_m1_idx next_m1_d next_m2_idx next_m2_d))))
  where
  jx = fst3 coords
  jy = snd3 coords
  jz = thd3 coords
  dx = jx - px
  d = distSq8 (px,py,pz) (jx,jy,jz)
  
  next_m1_idx = if d < m1_d then j else m1_idx
  next_m1_d   = if d < m1_d then d else m1_d
  next_m2_idx = if d < m1_d then m1_idx else (if d < m2_d then j else m2_idx)
  next_m2_d   = if d < m1_d then m1_d   else (if d < m2_d then d else m2_d)

scan_left px py pz [] m1_idx m1_d m2_idx m2_d = (m1_idx, m1_d, m2_idx, m2_d)
scan_left px py pz ((j, coords) : rest) m1_idx m1_d m2_idx m2_d =
  if dx * dx >= m2_d then (m1_idx, m1_d, m2_idx, m2_d)
  else seq next_m1_idx (seq next_m1_d (seq next_m2_idx (seq next_m2_d
       (scan_left px py pz rest next_m1_idx next_m1_d next_m2_idx next_m2_d))))
  where
  jx = fst3 coords
  jy = snd3 coords
  jz = thd3 coords
  dx = px - jx
  d = distSq8 (px,py,pz) (jx,jy,jz)
  
  next_m1_idx = if d < m1_d then j else m1_idx
  next_m1_d   = if d < m1_d then d else m1_d
  next_m2_idx = if d < m1_d then m1_idx else (if d < m2_d then j else m2_idx)
  next_m2_d   = if d < m1_d then m1_d   else (if d < m2_d then d else m2_d)

find_all_nearest_sweep lefts [] = []
find_all_nearest_sweep lefts ((i, coords) : rights) =
  (min i j1, max i j1, d1) : (min i j2, max i j2, d2) : find_all_nearest_sweep ((i, coords) : lefts) rights
  where
  px = fst3 coords
  py = snd3 coords
  pz = thd3 coords
  
  j_l = if lefts == [] then 0-1 else fst (hd lefts)
  d_l = if lefts == [] then 2000000000 else distSq8 (px,py,pz) (snd (hd lefts))
  
  j_r = if rights == [] then 0-1 else fst (hd rights)
  d_r = if rights == [] then 2000000000 else distSq8 (px,py,pz) (snd (hd rights))
  
  m1_idx_init = if d_l < d_r then j_l else j_r
  m1_d_init   = if d_l < d_r then d_l else d_r
  m2_idx_init = if d_l < d_r then j_r else j_l
  m2_d_init   = if d_l < d_r then d_r else d_l
  
  res_l = scan_left px py pz (if lefts == [] then [] else tl lefts) m1_idx_init m1_d_init m2_idx_init m2_d_init
  res_r = scan_right px py pz (if rights == [] then [] else tl rights) (fst4 res_l) (snd4 res_l) (thd4 res_l) (fth4 res_l)
  
  j1 = fst4 res_r
  d1 = snd4 res_r
  j2 = thd4 res_r
  d2 = fth4 res_r
  min a b = if a < b then a else b
  max a b = if a > b then a else b

dedup8 [] = []
dedup8 (x:xs) = if xs == [] then [x]
                else (if (fst3 x == fst3 y) & (snd3 x == snd3 y) then dedup8 (x : tl xs)
                      else x : dedup8 xs)
                where
                y = hd xs

find_root parent x =
  if p == x then (x, parent)
  else (root, h_insert root_parent x root)
  where
  p = h_lookup_def parent x x
  res = find_root parent p
  root = fst res
  root_parent = snd res

kruskal8 [] parent count = parent
kruskal8 (e:es) parent count =
  if count >= 1000 then parent
  else (if ru == rv then kruskal8 es parent_v count
        else kruskal8 es (h_insert parent_v ru rv) (count + 1))
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

group3 [] = []
group3 (x:y:z:rest) = (x, y, z) : group3 rest

solvePart1 input = s1 * s2 * s3
  where
  pts = group3 (parse_ints input)
  indexed_pts = zip ([0 .. length pts - 1], pts)
  sorted_pts = sort_pts indexed_pts
  edges = dedup8 (sort_edges [ e | e <- find_all_nearest_sweep [] sorted_pts; fst3 e >= 0 ])
  
  initial_parent = empty_map
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
  sorted_roots = sort_ints roots
  sizes = group_count sorted_roots
  sorted_sizes = reverse (sort_ints sizes)
  
  s1 = hd sorted_sizes
  s2 = hd (tl sorted_sizes)
  s3 = hd (tl (tl sorted_sizes))

main = "Advent of Code 2025 - Day 8 Results:\n" ++
       "  Part 1 (Circuit size product): " ++ show p1Result ++ "\n"
       where
       input = read "inputs/day8.txt"
       p1Result = solvePart1 input
