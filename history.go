// Package matchhistory 是战绩统计：胜率、连续胜场、乱序对局记录。
// 缺陷：平局被算进胜场，平局之后连胜没断，输入顺序被直接当成对局顺序。
package matchhistory

// Result 是一场对局的结果。
type Result int

// 对局结果。
const (
	Win Result = iota
	Draw
	Loss
)

// Match 是一场对局记录。
type Match struct {
	ID     string
	AtMs   int64
	Result Result
}

// Summary 是统计结果。
type Summary struct {
	Played        int
	Wins          int
	Draws         int
	Losses        int
	WinRate       float64 // 百分比，保留一位小数
	MaxStreak     int
	CurrentStreak int
}

// Summarize 汇总战绩。
// 缺陷：直接按输入顺序统计，平局走胜场分支，败局也不清连胜。
func Summarize(matches []Match) Summary {
	summary := Summary{Played: len(matches)}
	for _, match := range matches {
		if match.Result == Loss {
			summary.Losses++
			continue
		}
		summary.Wins++
		summary.CurrentStreak++
		if summary.CurrentStreak > summary.MaxStreak {
			summary.MaxStreak = summary.CurrentStreak
		}
	}
	if summary.Played > 0 {
		summary.WinRate = round1(float64(summary.Wins) * 100 / float64(summary.Played))
	}
	return summary
}

func round1(value float64) float64 {
	return float64(int64(value*10+0.5)) / 10
}
