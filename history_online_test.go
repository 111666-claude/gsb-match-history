package matchhistory

import (
	"math/rand"
	"testing"
)

// recompute 是独立参考实现：保留序列并逐场重算，窗口口径由调用方截取最近 RecentWindow 场。
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
		case Loss:
			summary.Losses++
			summary.CurrentStreak = 0
		}
	}
	if summary.Played > 0 {
		summary.WinRate = round1(float64(summary.Wins) * 100 / float64(summary.Played))
	}
	return summary
}

func repeatResults(n int, result Result) []Result {
	results := make([]Result, n)
	for index := range results {
		results[index] = result
	}
	return results
}

func TestDrawScoring(t *testing.T) {
	history := NewHistory()
	for _, result := range []Result{Win, Draw, Loss} {
		history.Record(Match{Result: result})
	}
	summary := history.Summary()
	if summary.Played != 3 || summary.Wins != 1 || summary.Draws != 1 || summary.Losses != 1 {
		t.Fatalf("平局应单独计分：%+v", summary)
	}
	if summary.WinRate != 33.3 || summary.MaxStreak != 1 || summary.CurrentStreak != 0 {
		t.Fatalf("一胜一平一负的口径不对：%+v", summary)
	}

	allDraws := NewHistory()
	for index := 0; index < 7; index++ {
		allDraws.Record(Match{Result: Draw})
	}
	drawSummary := allDraws.Summary()
	if drawSummary.Wins != 0 || drawSummary.Draws != 7 || drawSummary.Losses != 0 {
		t.Fatalf("全平不应有胜负：%+v", drawSummary)
	}
	if drawSummary.WinRate != 0 || drawSummary.MaxStreak != 0 || drawSummary.CurrentStreak != 0 {
		t.Fatalf("全平没有连胜：%+v", drawSummary)
	}

	history.Record(Match{Result: Win})
	if got := history.Summary(); got.Wins != 2 || got.Draws != 1 || got.Losses != 1 || got.WinRate != 50.0 {
		t.Fatalf("追加一胜后应为两胜一平一负：%+v", got)
	}
}

func TestStreakInterruptedByDrawAndLoss(t *testing.T) {
	history := NewHistory()
	sequence := []Result{Win, Win, Draw, Win, Win, Win, Loss, Win}
	for _, result := range sequence {
		history.Record(Match{Result: result})
	}
	summary := history.Summary()
	if summary.MaxStreak != 3 || summary.CurrentStreak != 1 {
		t.Fatalf("平局与败局都要中断连胜：%+v", summary)
	}
	if summary.Wins != 6 || summary.Draws != 1 || summary.Losses != 1 {
		t.Fatalf("中断场次计数不对：%+v", summary)
	}
}

func TestWindowBoundaryNoOffByOne(t *testing.T) {
	history := NewHistory()

	for index := 0; index < RecentWindow-1; index++ {
		history.Record(Match{Result: Win})
	}
	if recent := history.Recent(); recent.Played != RecentWindow-1 || recent.Wins != RecentWindow-1 {
		t.Fatalf("第 99 场时窗口应是全部 99 场：%+v", recent)
	}

	history.Record(Match{Result: Win})
	if recent := history.Recent(); recent.Played != RecentWindow || recent.Wins != RecentWindow ||
		recent.MaxStreak != RecentWindow || recent.CurrentStreak != RecentWindow {
		t.Fatalf("第 100 场时窗口应恰好 100 场全胜：%+v", recent)
	}

	history.Record(Match{Result: Loss})
	recent := history.Recent()
	if recent.Played != RecentWindow || recent.Wins != RecentWindow-1 || recent.Losses != 1 ||
		recent.MaxStreak != RecentWindow-1 || recent.CurrentStreak != 0 {
		t.Fatalf("第 101 场时最旧一胜应被挤出窗口：%+v", recent)
	}

	for index := 0; index < RecentWindow; index++ {
		history.Record(Match{Result: Loss})
	}
	recent = history.Recent()
	if recent.Played != RecentWindow || recent.Wins != 0 || recent.Losses != RecentWindow ||
		recent.MaxStreak != 0 || recent.CurrentStreak != 0 {
		t.Fatalf("窗口应只剩 100 场全负：%+v", recent)
	}

	summary := history.Summary()
	if summary.Played != 2*RecentWindow+1 || summary.Wins != RecentWindow || summary.Losses != RecentWindow+1 {
		t.Fatalf("全场统计不受窗口淘汰影响：%+v", summary)
	}
}

func TestWindowStreakAcrossBoundary(t *testing.T) {
	history := NewHistory()
	results := make([]Result, 0, RecentWindow+3)
	results = append(results, repeatResults(RecentWindow, Win)...)
	results = append(results, Loss, Win, Win)
	for _, result := range results {
		history.Record(Match{Result: result})
	}
	want := recompute(results[len(results)-RecentWindow:])
	if got := history.Recent(); got != want {
		t.Fatalf("跨窗口边界的连胜重算不一致：got %+v want %+v", got, want)
	}
}

func TestQueryCostDoesNotScanHistory(t *testing.T) {
	history := NewHistory()
	for index := 0; index < 200000; index++ {
		result := Win
		if index%3 == 0 {
			result = Loss
		}
		history.Record(Match{Result: result})
	}
	if history.Scanned() != 0 {
		t.Fatalf("Record 不应累计扫描计数：%d", history.Scanned())
	}
	for query := 0; query < 10000; query++ {
		history.Recent()
		history.Summary()
	}
	if scanned := history.Scanned(); scanned != 0 {
		t.Fatalf("查询一万次后 scanned 仍应为 0，实际 %d", scanned)
	}
	if recent := history.Recent(); recent.Played != RecentWindow {
		t.Fatalf("二十万场后窗口仍是 100 场：%+v", recent)
	}
}

func TestScaleMillionRecords(t *testing.T) {
	history := NewHistory()
	cycle := []Result{Win, Draw, Loss}
	lastWindow := make([]Result, 0, RecentWindow)
	for index := 0; index < 1000000; index++ {
		result := cycle[index%3]
		history.Record(Match{Result: result})
		lastWindow = append(lastWindow, result)
		if len(lastWindow) > RecentWindow {
			lastWindow = lastWindow[1:]
		}
	}
	if history.Len() != 1000000 {
		t.Fatalf("全场次计数不对：%d", history.Len())
	}
	wantRecent := recompute(lastWindow)
	for query := 0; query < 100000; query++ {
		if got := history.Recent(); got != wantRecent {
			t.Fatalf("第 %d 次查询窗口统计不一致：got %+v want %+v", query, got, wantRecent)
		}
	}
	if history.Scanned() != 0 {
		t.Fatalf("十万次查询不应扫描任何记录：%d", history.Scanned())
	}
	summary := history.Summary()
	if summary.Played != 1000000 || summary.Wins != 333334 || summary.Draws != 333333 || summary.Losses != 333333 {
		t.Fatalf("百万场全场计数不对：%+v", summary)
	}
	if summary.MaxStreak != 1 || summary.CurrentStreak != 1 || summary.WinRate != 33.3 {
		t.Fatalf("百万场连胜与胜率不对：%+v", summary)
	}
}

func assertInvariants(t *testing.T, label string, history *History, all []Result, step int) {
	t.Helper()

	full := history.Summary()
	if full != recompute(all) {
		t.Fatalf("case %s step %d 全场统计与重算不一致：got %+v want %+v", label, step, full, recompute(all))
	}
	window := all
	if len(window) > RecentWindow {
		window = window[len(window)-RecentWindow:]
	}
	wantRecent := recompute(window)
	if recent := history.Recent(); recent != wantRecent {
		t.Fatalf("case %s step %d 窗口统计与重算不一致：got %+v want %+v", label, step, recent, wantRecent)
	}

	for _, summary := range []Summary{full, history.Recent()} {
		if summary.Wins+summary.Draws+summary.Losses != summary.Played {
			t.Fatalf("case %s step %d wins+draws+losses != played：%+v", label, step, summary)
		}
		if summary.MaxStreak < summary.CurrentStreak {
			t.Fatalf("case %s step %d maxstreak < currentstreak：%+v", label, step, summary)
		}
	}
	if history.Recent().Played != len(window) || history.Len() != len(all) {
		t.Fatalf("case %s step %d 窗口/全场场次不对", label, step)
	}

	first := history.Summary()
	second := history.Summary()
	rFirst := history.Recent()
	rSecond := history.Recent()
	if first != second || rFirst != rSecond {
		t.Fatalf("case %s step %d 重复查询结果不稳定", label, step)
	}
	if history.Scanned() != 0 {
		t.Fatalf("case %s step %d scanned 应为 0：%d", label, step, history.Scanned())
	}
}

// fixedSequences 覆盖全平、全负、长连胜、窗口边界等确定场景。
func fixedSequences() [][]Result {
	sequences := [][]Result{
		{},
		{Win},
		{Draw},
		{Loss},
		{Win, Draw, Loss},
		{Win, Win, Win, Draw, Win},
	}
	for _, n := range []int{50, 99, 100, 101, 150, 205} {
		sequences = append(sequences, repeatResults(n, Draw))
		sequences = append(sequences, repeatResults(n, Loss))
	}
	sequences = append(sequences, repeatResults(250, Win))

	winThenBreak := repeatResults(RecentWindow, Win)
	winThenBreak = append(winThenBreak, Draw)
	sequences = append(sequences, winThenBreak)

	winThenLossWin := repeatResults(RecentWindow, Win)
	winThenLossWin = append(winThenLossWin, Loss, Win)
	sequences = append(sequences, winThenLossWin)

	lossesThenWins := append(repeatResults(20, Loss), repeatResults(RecentWindow, Win)...)
	sequences = append(sequences, lossesThenWins)

	alternating := make([]Result, 260)
	for index := range alternating {
		if index%2 == 0 {
			alternating[index] = Win
		} else {
			alternating[index] = Loss
		}
	}
	sequences = append(sequences, alternating)

	winDrawAlternating := make([]Result, 203)
	for index := range winDrawAlternating {
		if index%2 == 0 {
			winDrawAlternating[index] = Win
		} else {
			winDrawAlternating[index] = Draw
		}
	}
	sequences = append(sequences, winDrawAlternating)

	mixed := append(repeatResults(RecentWindow, Win), repeatResults(RecentWindow, Loss)...)
	mixed = append(mixed, repeatResults(5, Draw)...)
	sequences = append(sequences, mixed)

	return sequences
}

func randomSequence(random *rand.Rand) []Result {
	var length int
	switch random.Intn(4) {
	case 0:
		length = 95 + random.Intn(15) // 紧贴窗口边界
	case 1:
		length = random.Intn(RecentWindow + 1)
	default:
		length = random.Intn(260)
	}
	results := make([]Result, length)
	switch random.Intn(4) {
	case 0:
		for index := range results {
			results[index] = Result(random.Intn(3))
		}
	case 1:
		// 以随机长度结果段拼接，制造大量段淘汰与缩短。
		for index := 0; index < length; {
			result := Result(random.Intn(3))
			run := 1 + random.Intn(7)
			for offset := 0; offset < run && index < length; offset++ {
				results[index] = result
				index++
			}
		}
	case 2:
		// 长连胜里穿插少量中断。
		for index := range results {
			if index > 0 && index%37 == 0 {
				if random.Intn(2) == 0 {
					results[index] = Draw
				} else {
					results[index] = Loss
				}
			} else {
				results[index] = Win
			}
		}
	default:
		for index := range results {
			switch index % 3 {
			case 0:
				results[index] = Win
			case 1:
				results[index] = Draw
			default:
				results[index] = Loss
			}
		}
	}
	return results
}

func TestRandomSequencesMatchReference(t *testing.T) {
	fixed := fixedSequences()
	random := rand.New(rand.NewSource(20260930))
	total := 200
	for caseIndex := 0; caseIndex < total; caseIndex++ {
		var sequence []Result
		if caseIndex < len(fixed) {
			sequence = fixed[caseIndex]
		} else {
			sequence = randomSequence(random)
		}
		history := NewHistory()
		all := make([]Result, 0, len(sequence))
		for step, result := range sequence {
			history.Record(Match{ID: "m", AtMs: int64(step), Result: result})
			all = append(all, result)
			assertInvariants(t, "random", history, all, step)
		}
		// 空序列也要满足口径。
		if len(sequence) == 0 {
			assertInvariants(t, "empty", NewHistory(), nil, 0)
		}
	}
}
