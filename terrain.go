// Package terrain 是地形代价图：地形权重打底、障碍按覆盖层覆写、单格变更增量重算、
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
		for x := 0; x < w; x++ {
			m.terrain[[2]int{x, y}] = base
		}
	}
	m.build()
	return m, nil
}

// build 按当前地形铺一遍单格缓存与块和。
func (m *Map) build() {
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			key := [2]int{x, y}
			value, blocked := m.effective(x, y)
			m.cost[key] = value
			block := m.blockOf(x, y)
			m.blockSum[block] += value
			if blocked {
				m.blocked[key] = true
				m.blockBlocked[block]++
			}
		}
	}
}

// effective 算一格的有效代价：地形权重加非阻挡障碍的 Extra；阻挡障碍让该格不可通行。
func (m *Map) effective(x, y int) (int, bool) {
	value := m.weights[m.terrain[[2]int{x, y}]]
	if id, ok := m.overlay[[2]int{x, y}]; ok {
		ob := m.obstacles[id]
		if ob.Blocking {
			return value, true
		}
		value += ob.Extra
	}
	return value, false
}

// recomputeCell 只重算一格，并把差量同步进块和索引。
func (m *Map) recomputeCell(x, y int) {
	key := [2]int{x, y}
	block := m.blockOf(x, y)
	value, blocked := m.effective(x, y)
	m.blockSum[block] += value - m.cost[key]
	m.cost[key] = value
	if blocked != m.blocked[key] {
		if blocked {
			m.blocked[key] = true
			m.blockBlocked[block]++
		} else {
			delete(m.blocked, key)
			m.blockBlocked[block]--
		}
	}
	m.recomputed++
}

func (m *Map) blockOf(x, y int) [2]int { return [2]int{x / m.block, y / m.block} }

// inside 判断坐标是否在图内。
func (m *Map) inside(x, y int) bool { return x >= 0 && y >= 0 && x < m.w && y < m.h }

// SetTerrain 把一格改成另一种地形；只接受已登记的地形名。
func (m *Map) SetTerrain(x, y int, terrain string) bool {
	if !m.inside(x, y) {
		return false
	}
	if _, ok := m.weights[terrain]; !ok {
		return false
	}
	m.terrain[[2]int{x, y}] = terrain
	m.recomputeCell(x, y)
	return true
}

// PlaceObstacle 在一格上放一个障碍；只接受已登记的障碍 ID。
func (m *Map) PlaceObstacle(x, y int, id string) bool {
	if _, ok := m.obstacles[id]; !ok {
		return false
	}
	if !m.inside(x, y) {
		return false
	}
	m.overlay[[2]int{x, y}] = id
	m.recomputeCell(x, y)
	return true
}

// ClearObstacle 清掉一格上的障碍，返回该格原本是否有障碍。
func (m *Map) ClearObstacle(x, y int) bool {
	key := [2]int{x, y}
	if _, ok := m.overlay[key]; !ok {
		return false
	}
	delete(m.overlay, key)
	m.recomputeCell(x, y)
	return true
}

// Cost 返回一格的有效代价；不可通行或越界时 ok 为 false。
func (m *Map) Cost(x, y int) (int, bool) {
	if !m.inside(x, y) {
		return 0, false
	}
	m.scanned++
	key := [2]int{x, y}
	return m.cost[key], !m.blocked[key]
}

// RegionCost 返回矩形区域内每格有效代价之和；区域内出现不可通行格时 ok 为 false。
func (m *Map) RegionCost(x0, y0, x1, y1 int) (int, bool) {
	if !m.inside(x0, y0) || !m.inside(x1, y1) || x0 > x1 || y0 > y1 {
		return 0, false
	}
	sum := 0
	passable := true
	for by := y0 / m.block; by <= y1/m.block; by++ {
		for bx := x0 / m.block; bx <= x1/m.block; bx++ {
			block := [2]int{bx, by}
			m.scanned++
			cx0, cy0 := bx*m.block, by*m.block
			cx1 := min(cx0+m.block-1, m.w-1)
			cy1 := min(cy0+m.block-1, m.h-1)
			if x0 <= cx0 && cx1 <= x1 && y0 <= cy0 && cy1 <= y1 {
				sum += m.blockSum[block]
				if m.blockBlocked[block] > 0 {
					passable = false
				}
				continue
			}
			for y := max(y0, cy0); y <= min(y1, cy1); y++ {
				for x := max(x0, cx0); x <= min(x1, cx1); x++ {
					key := [2]int{x, y}
					if m.blocked[key] {
						passable = false
						continue
					}
					sum += m.cost[key]
				}
			}
		}
	}
	return sum, passable
}

// Recomputed 是累计重算的格子数（规模观测）。
func (m *Map) Recomputed() int { return m.recomputed }

// Scanned 是累计扫描量（规模观测）。
func (m *Map) Scanned() int { return m.scanned }
