// Package terrain 是地形代价图：地形权重打底、障碍按覆盖层叠加、单格变更增量重算、
// 区域代价按块和索引查询。
package terrain

import "errors"

// ErrBadMap 地图参数不合法。
var ErrBadMap = errors.New("地图参数不合法")

// Obstacle 是一种障碍定义。Blocking 为真时该格不可通行，否则 Extra 是叠加在地形权重上的额外代价。
type Obstacle struct {
	ID       string
	Blocking bool
	Extra    int
}

// Map 是地形代价图。
type Map struct {
	w, h, block  int
	weights      map[string]int
	obstacles    map[string]Obstacle
	terrain      map[[2]int]string
	overlay      map[[2]int]string
	cost         map[[2]int]int
	blocked      map[[2]int]bool
	blockSum     map[[2]int]int
	blockBlocked map[[2]int]int
	recomputed   int
	scanned      int
}

// New 建一张地形代价图。base 是所有格的初始地形，block 是区域查询索引的块边长。
func New(w, h, block int, base string, weights map[string]int, obstacles []Obstacle) (*Map, error) {
	if w <= 0 || h <= 0 || block <= 0 || len(weights) == 0 {
		return nil, ErrBadMap
	}
	m := &Map{
		w: w, h: h, block: block,
		weights:      map[string]int{},
		obstacles:    map[string]Obstacle{},
		terrain:      map[[2]int]string{},
		overlay:      map[[2]int]string{},
		cost:         map[[2]int]int{},
		blocked:      map[[2]int]bool{},
		blockSum:     map[[2]int]int{},
		blockBlocked: map[[2]int]int{},
	}
	for name, weight := range weights {
		if name == "" || weight <= 0 {
			return nil, ErrBadMap
		}
		m.weights[name] = weight
	}
	for _, ob := range obstacles {
		if ob.ID == "" {
			return nil, ErrBadMap
		}
		m.obstacles[ob.ID] = ob
	}
	if _, ok := m.weights[base]; !ok {
		return nil, ErrBadMap
	}
	for y := 0; y < h; y++ {
		for x := 0; x < m.w; x++ {
			m.terrain[[2]int{x, y}] = base
		}
	}
	m.build()
	return m, nil
}

// build 按当前地形铺一遍单格缓存与块和，不记账。
func (m *Map) build() {
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			key := [2]int{x, y}
			value := m.weights[m.terrain[key]]
			m.cost[key] = value
			m.blockSum[m.blockOf(x, y)] += value
		}
	}
}

func (m *Map) blockOf(x, y int) [2]int { return [2]int{x / m.block, y / m.block} }

// inside 判断坐标是否在图内。
func (m *Map) inside(x, y int) bool { return x >= 0 && y >= 0 && x < m.w && y < m.h }

// recompute 只重算单格并增量更新块和，记一次重算。
func (m *Map) recompute(x, y int) {
	key := [2]int{x, y}
	bkey := m.blockOf(x, y)
	if m.blocked[key] {
		delete(m.blocked, key)
		m.blockBlocked[bkey]--
		m.blockSum[bkey] -= m.cost[key]
	} else {
		m.blockSum[bkey] -= m.cost[key]
	}
	value := m.weights[m.terrain[key]]
	if id, ok := m.overlay[key]; ok {
		ob := m.obstacles[id]
		if ob.Blocking {
			m.blocked[key] = true
			m.blockBlocked[bkey]++
			m.cost[key] = 0
			m.recomputed++
			return
		}
		value += ob.Extra
	}
	m.cost[key] = value
	m.blockSum[bkey] += value
	m.recomputed++
}

// SetTerrain 把一格改成另一种已登记地形；未登记的地形名或越界坐标返回 false 且不改状态。
func (m *Map) SetTerrain(x, y int, terrain string) bool {
	if !m.inside(x, y) {
		return false
	}
	if _, ok := m.weights[terrain]; !ok {
		return false
	}
	key := [2]int{x, y}
	if m.terrain[key] == terrain {
		return true
	}
	m.terrain[key] = terrain
	m.recompute(x, y)
	return true
}

// PlaceObstacle 在一格上放一个已登记障碍；未登记 ID 或越界坐标返回 false 且不改状态。
func (m *Map) PlaceObstacle(x, y int, id string) bool {
	if !m.inside(x, y) {
		return false
	}
	if _, ok := m.obstacles[id]; !ok {
		return false
	}
	key := [2]int{x, y}
	if m.overlay[key] == id {
		return true
	}
	m.overlay[key] = id
	m.recompute(x, y)
	return true
}

// ClearObstacle 清掉一格上的障碍，返回该格原本是否有障碍。
func (m *Map) ClearObstacle(x, y int) bool {
	key := [2]int{x, y}
	if _, ok := m.overlay[key]; !ok {
		return false
	}
	delete(m.overlay, key)
	m.recompute(x, y)
	return true
}

// Cost 返回一格的有效代价；不可通行或越界时 ok 为 false。
func (m *Map) Cost(x, y int) (int, bool) {
	if !m.inside(x, y) {
		return 0, false
	}
	key := [2]int{x, y}
	if m.blocked[key] {
		return 0, false
	}
	return m.cost[key], true
}

// RegionCost 返回矩形区域内每格有效代价之和；区域内出现不可通行格时 ok 为 false。
// 查询走块和索引：完整落在区域内的块直接取块和，边界块逐格核对。
func (m *Map) RegionCost(x0, y0, x1, y1 int) (int, bool) {
	if !m.inside(x0, y0) || !m.inside(x1, y1) || x0 > x1 || y0 > y1 {
		return 0, false
	}
	bx0, by0 := x0/m.block, y0/m.block
	bx1, by1 := x1/m.block, y1/m.block
	sum := 0
	ok := true
	for by := by0; by <= by1; by++ {
		for bx := bx0; bx <= bx1; bx++ {
			fullX := bx > bx0 && bx < bx1
			fullY := by > by0 && by < by1
			bkey := [2]int{bx, by}
			m.scanned++
			if fullX && fullY {
				if m.blockBlocked[bkey] > 0 {
					ok = false
				} else {
					sum += m.blockSum[bkey]
				}
				continue
			}
			cx0, cy0 := bx*m.block, by*m.block
			cx1, cy1 := cx0+m.block-1, cy0+m.block-1
			if cx0 < x0 {
				cx0 = x0
			}
			if cy0 < y0 {
				cy0 = y0
			}
			if cx1 > x1 {
				cx1 = x1
			}
			if cy1 > y1 {
				cy1 = y1
			}
			if cx1 >= m.w {
				cx1 = m.w - 1
			}
			if cy1 >= m.h {
				cy1 = m.h - 1
			}
			for y := cy0; y <= cy1; y++ {
				for x := cx0; x <= cx1; x++ {
					key := [2]int{x, y}
					if m.blocked[key] {
						ok = false
					} else {
						sum += m.cost[key]
					}
				}
			}
		}
	}
	return sum, ok
}

// Recomputed 是累计重算的格子数（规模观测）。
func (m *Map) Recomputed() int { return m.recomputed }

// Scanned 是累计扫描量（规模观测）。
func (m *Map) Scanned() int { return m.scanned }
