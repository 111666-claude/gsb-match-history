// Package matchhistory 是战绩统计：全场统计随记录在线累积，最近 RecentWindow 场用滚动窗口维护。
// 缺陷：查询接口每次都重扫全量记录、平局被算进胜场、平局与败局都不清连胜、滚动窗口边界差一。
package matchhistory

// Result 是一场对局的结果。
type Result int

// 对局结果。
const (
	Win Result = iota
	Draw
	Loss
)

// RecentWindow 是滚动窗口的大小。
const RecentWindow = 100

// Match 是一场对局记录。
type Match struct {
	ID     string
	AtMs   int64
	Result Result
}

// Summary 是一段战绩的统计结果。WinRate 是百分比，保留一位小数。
type Summary struct {
	Played        int
	Wins          int
	Draws         int
	Losses        int
	WinRate       float64
	MaxStreak     int
	CurrentStreak int
}

// History 是战绩累计器。
type History struct {
	matches []Match
	scanned int
}

// NewHistory 建一个空的历史。
func NewHistory() *History {
	return &History{}
}

// Record 记一场对局，按调用顺序追加。
// 缺陷：只把记录追加进切片，全场与滚动窗口的统计都没在线维护。
func (h *History) Record(match Match) {
	h.matches = append(h.matches, match)
}

// Summary 是全场统计。
// 缺陷：每次调用都从头重扫全部记录；平局走胜场分支；败局不清连胜。
func (h *History) Summary() Summary {
	return summarize(h.matches, &h.scanned)
}

// Recent 是最近 RecentWindow 场的统计。
// 缺陷：窗口起点差一（多算一场），并且同样逐场重扫。
func (h *History) Recent() Summary {
	start := len(h.matches) - RecentWindow - 1
	if start < 0 {
		start = 0
	}
	return summarize(h.matches[start:], &h.scanned)
}

// Scanned 是查询接口累计扫过的记录数（规模观测）。
// 口径：Summary 与 Recent 必须是 O(1) 摊还，这个计数不应该随查询次数线性增长。
func (h *History) Scanned() int { return h.scanned }

// Len 是已记录的对局数。
func (h *History) Len() int { return len(h.matches) }

func summarize(matches []Match, scanned *int) Summary {
	summary := Summary{Played: len(matches)}
	for _, match := range matches {
		*scanned = *scanned + 1
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
