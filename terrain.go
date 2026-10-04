// Package terrain 是地形代价图：地形权重打底、障碍按覆盖层覆写、单格变更增量重算、
// 区域代价按块和索引查询。
// 缺陷：障碍覆盖层没生效、阻挡障碍没挡住、改了地形缓存不刷新、未知地形照收、
// 区域查询整图重算、增量重算没有记账。
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
	w, h, block int
	weights     map[string]int
	obstacles   map[string]Obstacle
	terrain     map[[2]int]string
	overlay     map[[2]int]string
	cost        map[[2]int]int
	blockSum    map[[2]int]int
	recomputed  int
	scanned     int
}

// New 建一张地形代价图。base 是所有格的初始地形，block 是区域查询索引的块边长。
func New(w, h, block int, base string, weights map[string]int, obstacles []Obstacle) (*Map, error) {
	if w <= 0 || h <= 0 || block <= 0 || len(weights) == 0 {
		return nil, ErrBadMap
	}
	m := &Map{
		w: w, h: h, block: block,
		weights:   map[string]int{},
		obstacles: map[string]Obstacle{},
		terrain:   map[[2]int]string{},
		overlay:   map[[2]int]string{},
		cost:      map[[2]int]int{},
		blockSum:  map[[2]int]int{},
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
			value := m.weights[m.terrain[[2]int{x, y}]]
			m.cost[[2]int{x, y}] = value
			m.blockSum[m.blockOf(x, y)] += value
		}
	}
}

func (m *Map) blockOf(x, y int) [2]int { return [2]int{x / m.block, y / m.block} }

// inside 判断坐标是否在图内。
func (m *Map) inside(x, y int) bool { return x >= 0 && y >= 0 && x < m.w && y < m.h }

// SetTerrain 把一格改成另一种地形。
func (m *Map) SetTerrain(x, y int, terrain string) bool {
	if !m.inside(x, y) {
		return false
	}
	m.terrain[[2]int{x, y}] = terrain
	return true
}

// PlaceObstacle 在一格上放一个障碍。
func (m *Map) PlaceObstacle(x, y int, id string) bool {
	if _, ok := m.obstacles[id]; !ok {
		return false
	}
	if !m.inside(x, y) {
		return false
	}
	m.overlay[[2]int{x, y}] = id
	return true
}

// ClearObstacle 清掉一格上的障碍，返回该格原本是否有障碍。
func (m *Map) ClearObstacle(x, y int) bool {
	key := [2]int{x, y}
	if _, ok := m.overlay[key]; !ok {
		return false
	}
	delete(m.overlay, key)
	return true
}

// Cost 返回一格的有效代价；不可通行或越界时 ok 为 false。
func (m *Map) Cost(x, y int) (int, bool) {
	if !m.inside(x, y) {
		return 0, false
	}
	m.scanned++
	return m.cost[[2]int{x, y}], true
}

// RegionCost 返回矩形区域内每格有效代价之和；区域内出现不可通行格时 ok 为 false。
func (m *Map) RegionCost(x0, y0, x1, y1 int) (int, bool) {
	if !m.inside(x0, y0) || !m.inside(x1, y1) || x0 > x1 || y0 > y1 {
		return 0, false
	}
	sum := 0
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			m.scanned++
			if x < x0 || x > x1 || y < y0 || y > y1 {
				continue
			}
			sum += m.cost[[2]int{x, y}]
		}
	}
	return sum, true
}

// Recomputed 是累计重算的格子数（规模观测）。
func (m *Map) Recomputed() int { return m.recomputed }

// Scanned 是累计扫描量（规模观测）。
func (m *Map) Scanned() int { return m.scanned }
