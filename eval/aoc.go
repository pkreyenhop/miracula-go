package eval

import (
	"sort"
	"strconv"
	"strings"
)

type aoc2PartialNode struct {
	input string
}

func (aoc2PartialNode) isNode() {}

type aoc11PartialNode struct {
	input string
}

func (aoc11PartialNode) isNode() {}

// Day 2
func Aoc2Solver(input string, part int) int {
	ranges := strings.Split(strings.TrimSpace(input), ",")
	total := 0
	for _, r := range ranges {
		parts := strings.Split(r, "-")
		if len(parts) != 2 {
			continue
		}
		start, _ := strconv.Atoi(parts[0])
		end, _ := strconv.Atoi(parts[1])
		for x := start; x <= end; x++ {
			if part == 1 {
				if checkPart1(x) {
					total += x
				}
			} else {
				if checkPart2(x) {
					total += x
				}
			}
		}
	}
	return total
}

func checkPart1(n int) bool {
	s := strconv.Itoa(n)
	if len(s)%2 != 0 {
		return false
	}
	half := len(s) / 2
	return s[:half] == s[half:]
}

func checkPart2(n int) bool {
	s := strconv.Itoa(n)
	L := len(s)
	for d := 1; d <= L/2; d++ {
		if L%d == 0 {
			chunk := s[:d]
			if strings.Repeat(chunk, L/d) == s {
				return true
			}
		}
	}
	return false
}

// Day 7
func Aoc7Solver(input string) int {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) == 0 {
		return 0
	}
	height := len(lines)
	// find S
	sCol := -1
	for c, char := range lines[0] {
		if char == 'S' {
			sCol = c
			break
		}
	}
	if sCol == -1 {
		return 0
	}

	active := map[int]bool{sCol: true}
	splits := 0

	for r := 1; r < height; r++ {
		nextActive := make(map[int]bool)
		row := lines[r]
		for c := range active {
			if c >= 0 && c < len(row) && row[c] == '^' {
				splits++
				nextActive[c-1] = true
				nextActive[c+1] = true
			} else {
				nextActive[c] = true
			}
		}
		active = nextActive
	}
	return splits
}

// Day 8
type Point3D struct {
	x, y, z int
}

type Edge struct {
	u, v   int
	distSq int
}

type UnionFind struct {
	parent []int
	size   []int
}

func NewUnionFind(n int) *UnionFind {
	parent := make([]int, n)
	size := make([]int, n)
	for i := range parent {
		parent[i] = i
		size[i] = 1
	}
	return &UnionFind{parent, size}
}

func (uf *UnionFind) Find(i int) int {
	if uf.parent[i] == i {
		return i
	}
	uf.parent[i] = uf.Find(uf.parent[i])
	return uf.parent[i]
}

func (uf *UnionFind) Union(i, j int) {
	rootI := uf.Find(i)
	rootJ := uf.Find(j)
	if rootI != rootJ {
		if uf.size[rootI] < uf.size[rootJ] {
			uf.parent[rootI] = rootJ
			uf.size[rootJ] += uf.size[rootI]
		} else {
			uf.parent[rootJ] = rootI
			uf.size[rootI] += uf.size[rootJ]
		}
	}
}

func Aoc8Solver(input string) int {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	var pts []Point3D
	for _, l := range lines {
		parts := strings.Split(l, ",")
		if len(parts) != 3 {
			continue
		}
		x, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		y, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		z, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
		pts = append(pts, Point3D{x, y, z})
	}

	n := len(pts)
	var edges []Edge
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			dx := pts[i].x - pts[j].x
			dy := pts[i].y - pts[j].y
			dz := pts[i].z - pts[j].z
			distSq := dx*dx + dy*dy + dz*dz
			edges = append(edges, Edge{i, j, distSq})
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		return edges[i].distSq < edges[j].distSq
	})

	uf := NewUnionFind(n)
	limit := 1000
	if len(edges) < limit {
		limit = len(edges)
	}
	for i := 0; i < limit; i++ {
		uf.Union(edges[i].u, edges[i].v)
	}

	compSizes := make(map[int]int)
	for i := 0; i < n; i++ {
		root := uf.Find(i)
		compSizes[root] = uf.size[root]
	}

	var sizes []int
	for _, s := range compSizes {
		sizes = append(sizes, s)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sizes)))

	prod := 1
	for i := 0; i < 3 && i < len(sizes); i++ {
		prod *= sizes[i]
	}
	return prod
}

// Day 9
type Point2D struct {
	x, y int
}

func Aoc9Solver(input string) int {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	var pts []Point2D
	for _, l := range lines {
		parts := strings.Split(l, ",")
		if len(parts) != 2 {
			continue
		}
		x, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		y, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		pts = append(pts, Point2D{x, y})
	}

	maxArea := 0
	for i := 0; i < len(pts); i++ {
		for j := 0; j < len(pts); j++ {
			if pts[j].x > pts[i].x && pts[j].y > pts[i].y {
				area := (pts[j].x - pts[i].x) * (pts[j].y - pts[i].y)
				if area > maxArea {
					maxArea = area
				}
			}
		}
	}
	return maxArea
}

// Day 10
func Aoc10Solver(input string) int {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	total := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		parts := strings.Split(l, " ")
		if len(parts) < 2 {
			continue
		}
		targetStr := parts[0]
		if len(targetStr) < 2 || targetStr[0] != '[' || targetStr[len(targetStr)-1] != ']' {
			continue
		}
		targetStr = targetStr[1 : len(targetStr)-1]
		target := make([]bool, len(targetStr))
		for i, c := range targetStr {
			target[i] = (c == '#')
		}

		var switches [][]int
		for _, p := range parts[1:] {
			if strings.HasPrefix(p, "(") && strings.HasSuffix(p, ")") {
				numsStr := p[1 : len(p)-1]
				numParts := strings.Split(numsStr, ",")
				var sw []int
				for _, ns := range numParts {
					val, _ := strconv.Atoi(strings.TrimSpace(ns))
					sw = append(sw, val)
				}
				switches = append(switches, sw)
			}
		}

		best := -1
		S := len(switches)
		for mask := 0; mask < (1 << S); mask++ {
			state := make([]bool, len(target))
			count := 0
			for i := 0; i < S; i++ {
				if (mask & (1 << i)) != 0 {
					count++
					for _, idx := range switches[i] {
						if idx >= 0 && idx < len(state) {
							state[idx] = !state[idx]
						}
					}
				}
			}
			match := true
			for i := range target {
				if state[i] != target[i] {
					match = false
					break
				}
			}
			if match {
				if best == -1 || count < best {
					best = count
				}
			}
		}
		if best != -1 {
			total += best
		}
	}
	return total
}

// Day 11
type Day11State struct {
	u      string
	hasDac bool
	hasFft bool
}

func Aoc11Solver(input string, part int) int {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	adj := make(map[string][]string)
	for _, l := range lines {
		parts := strings.Split(l, ":")
		if len(parts) != 2 {
			continue
		}
		u := strings.TrimSpace(parts[0])
		vStr := strings.TrimSpace(parts[1])
		vParts := strings.Fields(vStr)
		adj[u] = vParts
	}

	if part == 1 {
		memo := make(map[string]int)
		var dfs func(string) int
		dfs = func(u string) int {
			if u == "out" {
				return 1
			}
			if v, ok := memo[u]; ok {
				return v
			}
			sum := 0
			for _, next := range adj[u] {
				sum += dfs(next)
			}
			memo[u] = sum
			return sum
		}
		return dfs("you")
	} else {
		memo := make(map[Day11State]int)
		var dfs func(string, bool, bool) int
		dfs = func(u string, hasDac, hasFft bool) int {
			if u == "dac" {
				hasDac = true
			}
			if u == "fft" {
				hasFft = true
			}
			if u == "out" {
				if hasDac && hasFft {
					return 1
				}
				return 0
			}
			state := Day11State{u, hasDac, hasFft}
			if v, ok := memo[state]; ok {
				return v
			}
			sum := 0
			for _, next := range adj[u] {
				sum += dfs(next, hasDac, hasFft)
			}
			memo[state] = sum
			return sum
		}
		return dfs("svr", false, false)
	}
}
