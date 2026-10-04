// Command terrain-cost 跑地形代价图样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"example.com/terrain"
)

func newMap(w, h, block int) (*terrain.Map, error) {
	weights := map[string]int{"grass": 1, "rock": 5}
	obstacles := []terrain.Obstacle{
		{ID: "wall", Blocking: true},
		{ID: "mud", Extra: 3},
	}
	return terrain.New(w, h, block, "grass", weights, obstacles)
}

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("terrain-cost", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "overlay", "overlay / slow / terrain / unknown / stale / blocked / scan")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *sample == "overlay" {
		m, err := newMap(4, 4, 2)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		m.PlaceObstacle(1, 1, "wall")
		cost, ok := m.Cost(1, 1)
		fmt.Fprintf(stdout, "ok=%v cost=%d\n", ok, cost)
		return 0
	}
	if *sample == "slow" {
		m, err := newMap(4, 4, 2)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		m.PlaceObstacle(2, 2, "mud")
		cost, ok := m.Cost(2, 2)
		fmt.Fprintf(stdout, "ok=%v cost=%d\n", ok, cost)
		return 0
	}
	if *sample == "terrain" {
		m, err := newMap(4, 4, 2)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		m.SetTerrain(0, 0, "rock")
		cost, ok := m.Cost(0, 0)
		fmt.Fprintf(stdout, "ok=%v cost=%d\n", ok, cost)
		return 0
	}
	if *sample == "unknown" {
		m, err := newMap(4, 4, 2)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "ok=%v\n", m.SetTerrain(0, 0, "lava"))
		return 0
	}
	if *sample == "stale" {
		m, err := newMap(4, 4, 2)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		m.SetTerrain(0, 0, "rock")
		sum, ok := m.RegionCost(0, 0, 0, 0)
		fmt.Fprintf(stdout, "sum=%d ok=%v\n", sum, ok)
		return 0
	}
	if *sample == "blocked" {
		m, err := newMap(4, 4, 2)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		m.PlaceObstacle(1, 1, "wall")
		sum, ok := m.RegionCost(0, 0, 1, 1)
		fmt.Fprintf(stdout, "sum=%d ok=%v\n", sum, ok)
		return 0
	}
	if *sample == "scan" {
		m, err := newMap(60, 60, 8)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		for index := 0; index < 3000; index++ {
			m.RegionCost(0, 0, 1, 1)
		}
		for index := 0; index < 3000; index++ {
			m.SetTerrain(index%60, (index/60)%60, "rock")
		}
		fmt.Fprintf(stdout, "scanned=%d recomputed=%d\n", m.Scanned(), m.Recomputed())
		return 0
	}
	fmt.Fprintln(stderr, "需要 --sample overlay|slow|terrain|unknown|stale|blocked|scan")
	return 2
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
