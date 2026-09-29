# match-history

战绩统计：胜率、最长连胜、当前连胜，只用 Go 标准库。

```
go test ./...
go run ./cmd/match-history --sample draw
go run ./cmd/match-history --sample streak
go run ./cmd/match-history --sample unsorted
```

## 口径（README 为准）

- **平局既不是胜场也不是败场**：`draws` 必须单独统计，`winrate = wins / played`（平局占分母
  不占分子，保留一位小数）。`--sample draw`（一胜一平一负）按口径是
  `wins=1 draws=1 losses=1 winrate=33.3`，现在是 `wins=2 draws=0 winrate=66.7`。
- **平局要断连胜**：只有连续的 `Win` 才算连胜，`Draw` 与 `Loss` 都清零 `currentstreak`；
  `maxstreak` 是整段历史里最长的那次。`--sample streak`（三连胜、平、胜）按口径是
  `maxstreak=3 currentstreak=1`，现在是 `maxstreak=5 currentstreak=5`。
- **按时间顺序统计**：输入可能乱序，必须按 `AtMs` 升序处理，`AtMs` 相同则按 `ID` 字典序升序。
  `--sample unsorted`（与 draw 同样的三场、顺序打乱）按口径输出与 `draw` 完全一致，
  现在会得到 `winrate=66.7 maxstreak=2`。
- 不变量：`wins + draws + losses == played`；同一批对局任意打乱输入顺序结果不变；
  `maxstreak >= currentstreak`；重复统计结果相同。
- 规模：一次 100 万场对局，O(n log n) 以内处理完（排序 + 单遍扫描），内存 O(n) 量级；
  不许把每场对局的状态快照都留下。

## 现在的行为

- `Summarize` 只在 `Loss` 分支加败场，`Win` 和 `Draw` 都进胜场分支，`Draws` 永远是 0。
- 败局 `continue` 掉，`CurrentStreak` 从没被清零，最长连胜会被算大。
- 直接把输入顺序当成对局顺序，乱序输入结果就错。

## 输出

```
played=3 wins=1 draws=1 losses=1 winrate=33.3 maxstreak=1 currentstreak=0
played=5 wins=4 draws=1 losses=0 winrate=80.0 maxstreak=3 currentstreak=1
played=3 wins=1 draws=1 losses=1 winrate=33.3 maxstreak=1 currentstreak=0
```

## 目录

```
history.go              战绩统计
cmd/match-history       命令行入口
history_test.go         go test 用例
```
