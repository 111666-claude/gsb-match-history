# match-history

战绩统计：全场统计在线累积，最近 100 场用滚动窗口维护，只用 Go 标准库。

```
go test ./...
go run ./cmd/match-history --sample draw
go run ./cmd/match-history --sample streak
go run ./cmd/match-history --sample recent
go run ./cmd/match-history --sample cost
```

## 口径（README 为准）

- **在线累积**：`Record` 之后 `Summary()` 与 `Recent()` 都必须是就地更新后的结果，
  查询接口不许再遍历历史记录；`Scanned()` 统计查询期间扫过的记录数，不许随查询次数增长。
- **全场统计**：`played` 是记录总数；`winrate` 等于 `wins / played * 100`，保留一位小数。
- **平局**：`Draw` 既不是胜场也不是败局，单独计入 `draws`，并中断连胜。
- **连胜**：只有连续的 `Win` 才算连胜；`maxstreak` 是整段历史里最长的一次，
  `currentstreak` 是结尾连续胜场数。
- **滚动窗口**：`Recent()` 只看最近 `RecentWindow = 100` 场（不足 100 场时就是全部），
  窗口内的 `played`/`wins`/`maxstreak` 都要按这 100 场单独算，与"把这 100 场重建一遍"一致。
- **规模**：单次 100 万场、每秒 10 万次查询，`Record` 与查询都摊还 O(1)，内存 O(1)
  （只保留最近 100 场的环形缓冲）；不许保留全量记录再重算。

## 输出契约（不改格式）

```
wins=1 draws=1 losses=1 winrate=33.3 maxstreak=1 currentstreak=0
maxstreak=3 currentstreak=1
played=100 wins=100 losses=0 winrate=100.0 maxstreak=100
scanned=0
```

## 常量

```
RecentWindow = 100
```

## 目录

```
history.go              战绩统计
cmd/match-history       命令行入口
history_test.go         go test 用例
```
