package matchhistory

import "testing"

func TestAllWins(t *testing.T) {
	summary := Summarize([]Match{
		{ID: "a", AtMs: 1000, Result: Win},
		{ID: "b", AtMs: 2000, Result: Win},
		{ID: "c", AtMs: 3000, Result: Win},
	})
	if summary.Played != 3 || summary.Wins != 3 || summary.Losses != 0 {
		t.Fatalf("三连胜的场次不对：%+v", summary)
	}
	if summary.WinRate != 100 {
		t.Fatalf("三连胜的胜率应该是 100.0：%v", summary.WinRate)
	}
	if summary.MaxStreak != 3 || summary.CurrentStreak != 3 {
		t.Fatalf("三连胜的连胜数不对：%+v", summary)
	}
}

func TestEmptyHistory(t *testing.T) {
	summary := Summarize(nil)
	if summary.Played != 0 || summary.WinRate != 0 {
		t.Fatalf("空战绩应该是 0：%+v", summary)
	}
}

func TestWinRateKeepsOneDecimal(t *testing.T) {
	summary := Summarize([]Match{
		{ID: "a", AtMs: 1000, Result: Win},
		{ID: "b", AtMs: 2000, Result: Loss},
		{ID: "c", AtMs: 3000, Result: Loss},
	})
	if summary.WinRate != 33.3 {
		t.Fatalf("一胜两负的胜率应该是 33.3：%v", summary.WinRate)
	}
}
