package matchhistory

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

// recompute 是参考实现：把给定序列从头重算一遍（只在测试里用）。
func recompute(results []Result) Summary {
	summary := Summary{Played: len(results)}
	for _, result := range results {
		switch result {
		case Win:
			summary.Wins++
			summary.CurrentStreak++
			if summary.CurrentStreak > summary.MaxStreak {
				summary.MaxStreak = summary.CurrentStreak
			}
		case Draw:
			summary.Draws++
			summary.CurrentStreak = 0
		default:
			summary.Losses++
			summary.CurrentStreak = 0
		}
	}
	if summary.Played > 0 {
		summary.WinRate = math.Round(float64(summary.Wins)/float64(summary.Played)*1000) / 10
	}
	return summary
}

func recordResult(history *History, index int, result Result) {
	history.Record(Match{ID: fmt.Sprintf("m-%d", index), AtMs: int64(index), Result: result})
}

// TestDrawScoring 平局既不是胜场也不是败局，并中断连胜；胜率只按胜场算且保留一位小数。
func TestDrawScoring(t *testing.T) {
	history := NewHistory()
	recordResult(history, 0, Win)
	recordResult(history, 1, Draw)
	recordResult(history, 2, Loss)
	summary := history.Summary()
	if summary.Played != 3 || summary.Wins != 1 || summary.Draws != 1 || summary.Losses != 1 {
		t.Fatalf("一胜一平一负的场次不对：%+v", summary)
	}
	if summary.WinRate != 33.3 {
		t.Fatalf("一胜一平一负的胜率应该是 33.3：%+v", summary)
	}
	if summary.MaxStreak != 1 || summary.CurrentStreak != 0 {
		t.Fatalf("平局必须中断连胜：%+v", summary)
	}
}

// TestStreakInterruptedByDrawAndLoss 平局与败局都要中断连胜，最长连胜不被续上。
func TestStreakInterruptedByDrawAndLoss(t *testing.T) {
	history := NewHistory()
	for index := 0; index < 3; index++ {
		recordResult(history, index, Win)
	}
	recordResult(history, 3, Draw)
	if summary := history.Summary(); summary.MaxStreak != 3 || summary.CurrentStreak != 0 {
		t.Fatalf("三连胜后平局：%+v", summary)
	}
	recordResult(history, 4, Win)
	recordResult(history, 5, Win)
	recordResult(history, 6, Loss)
	recordResult(history, 7, Win)
	summary := history.Summary()
	if summary.MaxStreak != 3 || summary.CurrentStreak != 1 {
		t.Fatalf("败局必须中断连胜且最长连胜不被续长：%+v", summary)
	}
	if summary.Wins != 6 || summary.Draws != 1 || summary.Losses != 1 || summary.Played != 8 {
		t.Fatalf("中断序列场次不对：%+v", summary)
	}
}

// TestWindowBoundary 滚动窗口正好是最近 100 场，边界不许差一。
func TestWindowBoundary(t *testing.T) {
	history := NewHistory()
	// 99 胜 + 1 负：恰好填满窗口。
	for index := 0; index < 99; index++ {
		recordResult(history, index, Win)
	}
	recordResult(history, 99, Loss)
	recent := history.Recent()
	if recent.Played != 100 || recent.Wins != 99 || recent.Losses != 1 || recent.MaxStreak != 99 || recent.CurrentStreak != 0 {
		t.Fatalf("第 100 场后窗口应为 99 胜 1 负：%+v", recent)
	}

	// 再来一胜：第 1 场（胜）被淘汰，最近 100 场是 98 胜 + 负 + 胜。
	recordResult(history, 100, Win)
	recent = history.Recent()
	if recent.Played != 100 || recent.Wins != 99 || recent.Losses != 1 {
		t.Fatalf("第 101 场后窗口仍应正好 100 场：%+v", recent)
	}
	if recent.MaxStreak != 98 || recent.CurrentStreak != 1 {
		t.Fatalf("窗口跨过边界后连胜段应重算：%+v", recent)
	}

	// 20 负 + 100 胜：最近 100 场全胜。
	history = NewHistory()
	for index := 0; index < 20; index++ {
		recordResult(history, index, Loss)
	}
	for index := 20; index < 120; index++ {
		recordResult(history, index, Win)
	}
	recent = history.Recent()
	if recent.Played != 100 || recent.Wins != 100 || recent.Losses != 0 || recent.WinRate != 100 || recent.MaxStreak != 100 {
		t.Fatalf("最近 100 场应全胜：%+v", recent)
	}
}

// TestQueryCost 查询不遍历历史，Scanned 不随查询增长，重复查询结果相同。
func TestQueryCost(t *testing.T) {
	history := NewHistory()
	for index := 0; index < 200000; index++ {
		result := Win
		if index%3 == 0 {
			result = Loss
		}
		recordResult(history, index, result)
	}
	if history.Scanned() != 0 {
		t.Fatalf("Record 阶段不应累计扫描数：%d", history.Scanned())
	}
	firstSummary := history.Summary()
	firstRecent := history.Recent()
	for query := 0; query < 10000; query++ {
		if got := history.Summary(); got != firstSummary {
			t.Fatalf("第 %d 次 Summary 结果不一致：%+v != %+v", query, got, firstSummary)
		}
		if got := history.Recent(); got != firstRecent {
			t.Fatalf("第 %d 次 Recent 结果不一致：%+v != %+v", query, got, firstRecent)
		}
	}
	if history.Scanned() != 0 {
		t.Fatalf("一万次查询后 scanned 必须仍是 0：%d", history.Scanned())
	}
	if firstRecent.Played != RecentWindow {
		t.Fatalf("最近窗口场次应为 %d：%+v", RecentWindow, firstRecent)
	}
}

// TestInvariants 每一步都满足：wins+draws+losses=played；maxstreak>=currentstreak。
func TestInvariants(t *testing.T) {
	history := NewHistory()
	sequence := []Result{Win, Win, Draw, Win, Loss, Loss, Win, Draw, Draw, Win, Win, Win, Loss}
	for index, result := range sequence {
		recordResult(history, index, result)
		for name, summary := range map[string]Summary{"summary": history.Summary(), "recent": history.Recent()} {
			if summary.Wins+summary.Draws+summary.Losses != summary.Played {
				t.Fatalf("%s 场次不守恒（第 %d 场后）：%+v", name, index, summary)
			}
			if summary.MaxStreak < summary.CurrentStreak {
				t.Fatalf("%s 最长连胜小于当前连胜（第 %d 场后）：%+v", name, index, summary)
			}
		}
	}
}

// TestRandomCrossCheck 200 组随机/边界序列，全场与窗口统计逐项对照“独立重算”的参考实现。
func TestRandomCrossCheck(t *testing.T) {
	sequences := make([][]Result, 0, 200)
	// 含 README 点名的边界形态：全平、全负、长连胜、跨窗口边界。
	allSame := func(length int, result Result) []Result {
		sequence := make([]Result, length)
		for index := range sequence {
			sequence[index] = result
		}
		return sequence
	}
	sequences = append(sequences,
		allSame(150, Draw),
		allSame(150, Loss),
		allSame(250, Win),
	)
	crossing := make([]Result, 250)
	for index := range crossing { // 每 99 场一个平局，长连胜跨窗口边界被切。
		if index%99 == 98 {
			crossing[index] = Draw
		} else {
			crossing[index] = Win
		}
	}
	sequences = append(sequences, crossing)

	random := rand.New(rand.NewSource(20260930))
	for len(sequences) < 200 {
		length := random.Intn(260) // 0..259，覆盖空序列、窗口内、一到两个多窗口。
		sequence := make([]Result, length)
		for index := range sequence {
			sequence[index] = Result(random.Intn(3))
		}
		sequences = append(sequences, sequence)
	}

	for caseIndex, sequence := range sequences {
		history := NewHistory()
		for index, result := range sequence {
			recordResult(history, index, result)

			wantSummary := recompute(sequence[:index+1])
			gotSummary := history.Summary()
			if !summariesEqual(gotSummary, wantSummary) {
				t.Fatalf("case %d 第 %d 场后全场统计不一致：got=%+v want=%+v", caseIndex, index, gotSummary, wantSummary)
			}

			left := index + 1 - RecentWindow
			if left < 0 {
				left = 0
			}
			wantRecent := recompute(sequence[left : index+1])
			gotRecent := history.Recent()
			if !summariesEqual(gotRecent, wantRecent) {
				t.Fatalf("case %d 第 %d 场后窗口统计不一致：got=%+v want=%+v", caseIndex, index, gotRecent, wantRecent)
			}

			if gotSummary.Wins+gotSummary.Draws+gotSummary.Losses != gotSummary.Played ||
				gotRecent.Wins+gotRecent.Draws+gotRecent.Losses != gotRecent.Played {
				t.Fatalf("case %d 第 %d 场后场次不守恒", caseIndex, index)
			}
			if gotSummary.MaxStreak < gotSummary.CurrentStreak || gotRecent.MaxStreak < gotRecent.CurrentStreak {
				t.Fatalf("case %d 第 %d 场后连胜不变量被破坏", caseIndex, index)
			}
			if history.Scanned() != 0 {
				t.Fatalf("case %d 第 %d 场后 scanned 应为 0：%d", caseIndex, index, history.Scanned())
			}
		}
		// 同一终局重复查询必须完全相同。
		if got := history.Recent(); got != history.Recent() || history.Summary() != history.Summary() {
			t.Fatalf("case %d 重复查询结果不稳定", caseIndex)
		}
	}
}

func summariesEqual(got, want Summary) bool {
	return got.Played == want.Played &&
		got.Wins == want.Wins &&
		got.Draws == want.Draws &&
		got.Losses == want.Losses &&
		got.MaxStreak == want.MaxStreak &&
		got.CurrentStreak == want.CurrentStreak &&
		math.Abs(got.WinRate-want.WinRate) < 1e-9
}

// TestScaleOneMillion 单次 100 万场后高频查询：结果正确、不扫历史。
func TestScaleOneMillion(t *testing.T) {
	if testing.Short() {
		t.Skip("规模用例")
	}
	history := NewHistory()
	for index := 0; index < 1000000; index++ {
		result := Result(index % 3)
		recordResult(history, index, result)
	}
	summary := history.Summary()
	if summary.Played != 1000000 {
		t.Fatalf("总场次不对：%+v", summary)
	}
	if summary.Wins+summary.Draws+summary.Losses != 1000000 {
		t.Fatalf("场次不守恒：%+v", summary)
	}
	recent := history.Recent()
	if recent.Played != RecentWindow {
		t.Fatalf("窗口场次不对：%+v", recent)
	}
	for query := 0; query < 100000; query++ {
		if history.Recent() != recent || history.Summary() != summary {
			t.Fatalf("第 %d 次查询结果漂移", query)
		}
	}
	if history.Scanned() != 0 {
		t.Fatalf("十万次查询后 scanned 应为 0：%d", history.Scanned())
	}
}

func BenchmarkRecord(b *testing.B) {
	history := NewHistory()
	for index := 0; index < b.N; index++ {
		recordResult(history, index, Result(index%3))
	}
}

func BenchmarkRecent(b *testing.B) {
	history := NewHistory()
	for index := 0; index < RecentWindow*10; index++ {
		recordResult(history, index, Result(index%3))
	}
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		_ = history.Recent()
	}
}
