package matchhistory

import "testing"

func TestAllWins(t *testing.T) {
	history := NewHistory()
	for index := 0; index < 3; index++ {
		history.Record(Match{ID: string(rune('a' + index)), AtMs: int64(index * 1000), Result: Win})
	}
	summary := history.Summary()
	if summary.Played != 3 || summary.Wins != 3 || summary.Losses != 0 || summary.Draws != 0 {
		t.Fatalf("三连胜的场次不对：%+v", summary)
	}
	if summary.WinRate != 100 || summary.MaxStreak != 3 || summary.CurrentStreak != 3 {
		t.Fatalf("三连胜的胜率与连胜数不对：%+v", summary)
	}
}

func TestEmptyHistory(t *testing.T) {
	summary := NewHistory().Summary()
	if summary.Played != 0 || summary.WinRate != 0 {
		t.Fatalf("空战绩应该是 0：%+v", summary)
	}
}

func TestWinRateKeepsOneDecimal(t *testing.T) {
	history := NewHistory()
	history.Record(Match{ID: "a", AtMs: 1000, Result: Win})
	history.Record(Match{ID: "b", AtMs: 2000, Result: Loss})
	history.Record(Match{ID: "c", AtMs: 3000, Result: Loss})
	if got := history.Summary().WinRate; got != 33.3 {
		t.Fatalf("一胜两负的胜率应该是 33.3：%v", got)
	}
}

func TestRecentOnShortHistory(t *testing.T) {
	history := NewHistory()
	for index := 0; index < 3; index++ {
		history.Record(Match{ID: string(rune('a' + index)), AtMs: int64(index), Result: Win})
	}
	recent := history.Recent()
	if recent.Played != 3 || recent.Wins != 3 {
		t.Fatalf("不足一个窗口时最近战绩就是全部：%+v", recent)
	}
}

func TestConstants(t *testing.T) {
	if RecentWindow != 100 {
		t.Fatal("滚动窗口常量被改了")
	}
}
