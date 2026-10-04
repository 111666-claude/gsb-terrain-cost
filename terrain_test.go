package terrain

import "testing"

func TestNewRejectsBadMap(t *testing.T) {
	if _, err := New(0, 4, 2, "grass", map[string]int{"grass": 1}, nil); err != ErrBadMap {
		t.Fatal("尺寸或权重不合法都应该报 ErrBadMap")
	}
}

func TestCostOfUntouchedTile(t *testing.T) {
	m, err := New(4, 4, 2, "grass", map[string]int{"grass": 1, "rock": 5}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cost, ok := m.Cost(0, 0); !ok || cost != 1 {
		t.Fatalf("没动过的草地代价应该是 1，得到 cost=%d ok=%v", cost, ok)
	}
}

func TestCostOutOfRange(t *testing.T) {
	m, err := New(4, 4, 2, "grass", map[string]int{"grass": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Cost(4, 0); ok {
		t.Fatal("越界坐标应该返回 false")
	}
}

func TestRegionMatchesCostSum(t *testing.T) {
	m, err := New(4, 4, 2, "grass", map[string]int{"grass": 1, "rock": 5}, nil)
	if err != nil {
		t.Fatal(err)
	}
	sum := 0
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			value, _ := m.Cost(x, y)
			sum += value
		}
	}
	got, ok := m.RegionCost(0, 0, 1, 1)
	if !ok || got != sum {
		t.Fatalf("区域代价应该等于逐格之和，得到 sum=%d ok=%v，期望 %d", got, ok, sum)
	}
}

func TestCountersStartAtZero(t *testing.T) {
	m, err := New(4, 4, 2, "grass", map[string]int{"grass": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.Scanned() != 0 || m.Recomputed() != 0 {
		t.Fatal("两个计数器初始都应该是 0")
	}
}
