# terrain-cost

地形代价图：地形权重打底、障碍按覆盖层覆写、单格变更增量重算、区域代价按块和索引查询，
只用 Go 标准库。

```
go run ./cmd/terrain-cost --sample=overlay
go run ./cmd/terrain-cost --sample=slow
go run ./cmd/terrain-cost --sample=terrain
go run ./cmd/terrain-cost --sample=unknown
go run ./cmd/terrain-cost --sample=stale
go run ./cmd/terrain-cost --sample=blocked
go run ./cmd/terrain-cost --sample=scan
go test ./...
go vet ./...
```

## 口径（README 为准）

- **有效代价**：每格代价等于该格地形权重加上障碍的 `Extra`；障碍是覆盖在地形之上的层。
- **阻挡**：`Blocking` 障碍所在的格不可通行，`Cost` 返回 `ok=false`。
- **拒绝**：`SetTerrain` 只接受已登记的地形名，`PlaceObstacle` 只接受已登记的障碍 ID，
  不合法一律返回 `false` 且不改任何状态。
- **区域**：`RegionCost` 返回区域内每格有效代价之和；区域内出现不可通行格时返回 `ok=false`；
  区域不合法（越界或 `x0>x1`）也返回 `ok=false`。
- **增量**：单格变更只重算该格，`Recomputed` 的总增量不超过变更次数。
- **索引**：区域查询走块和索引，`Scanned` 单次不超过覆盖到的块数加边界格数，不随整图面积增长。
- **不变量**：没有不可通行格时 `RegionCost` 等于区域内逐格 `Cost` 之和；同一串变更重复回放结果相同。
- **规模**：60×60 整图、每秒 10 万次区域查询，单次摊还 O(覆盖块数 加 块边长)。

## 输出契约（不改格式）

```
ok=.. cost=..
ok=..
sum=.. ok=..
scanned=.. recomputed=..
```

## 目录

```
terrain.go            权重、覆盖层、增量缓存与块和索引
terrain_test.go       单元测试
cmd/terrain-cost/     命令行入口
```
