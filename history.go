// Package matchhistory 是战绩统计：全场统计随记录在线累积，最近 RecentWindow 场用滚动窗口维护。
// 只保留最近 RecentWindow 场的环形缓冲，Record 与查询均摊 O(1)，内存 O(1)。
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

// winRun 是窗口内一段连续结果；只对 Win 段统计连胜长度。
type winRun struct {
	result Result
	length int
}

// History 是战绩累计器。
// 全场统计与窗口统计都在 Record 时就地更新，查询只读已有状态。
type History struct {
	// 全场统计。
	played        int
	wins          int
	draws         int
	losses        int
	maxStreak     int
	currentStreak int

	// 最近 RecentWindow 场的环形缓冲（只存结果，不保留全量历史）。
	ring     [RecentWindow]Result
	ringPos  int // 下一个写入位置
	ringFull bool

	// 窗口内按顺序切分的连续结果段，环形存放，最多 RecentWindow 个。
	runs    [RecentWindow]winRun
	runsPos int // 下一个写入位置
	runsLen int // 现存段数

	// winLenCount[L] 是窗口内长度为 L 的连胜段个数；winMaxLen 是当前最长连胜段。
	// 连胜段长度只增不减时最大值 O(1) 上升；缩短或淘汰时借助计数下移，
	// 每次下移对应此前的一次增长，整体摊还 O(1)。
	winLenCount [RecentWindow + 1]int
	winMaxLen   int

	// 窗口内各结果计数。
	wWins   int
	wDraws  int
	wLosses int
}

// NewHistory 建一个空的历史。
func NewHistory() *History {
	return &History{}
}

// addWinLen 登记一个长度为 length 的连胜段出现。
func (h *History) addWinLen(length int) {
	if length > RecentWindow {
		length = RecentWindow
	}
	h.winLenCount[length]++
	if length > h.winMaxLen {
		h.winMaxLen = length
	}
}

// removeWinLen 登记一个长度为 length 的连胜段消失；必要时下调最长连胜。
func (h *History) removeWinLen(length int) {
	if length <= 0 || length > RecentWindow {
		return
	}
	h.winLenCount[length]--
	for h.winMaxLen > 0 && h.winLenCount[h.winMaxLen] == 0 {
		h.winMaxLen--
	}
}

// growLastRun 把窗口最后一段（必为连胜段）延长一场。
func (h *History) growLastRun() {
	index := (h.runsPos - 1 + RecentWindow) % RecentWindow
	old := h.runs[index].length
	h.removeWinLen(old)
	h.runs[index].length = old + 1
	h.addWinLen(old + 1)
}

// appendRun 在窗口尾部追加一个长度为 1 的结果段。
func (h *History) appendRun(result Result) {
	h.runs[h.runsPos] = winRun{result: result, length: 1}
	h.runsPos = (h.runsPos + 1) % RecentWindow
	h.runsLen++
	if result == Win {
		h.addWinLen(1)
	}
}

// evictFront 淘汰窗口最前一场，并同步缩短或移除第一段。
func (h *History) evictFront() {
	front := (h.runsPos - h.runsLen + RecentWindow) % RecentWindow
	run := h.runs[front]
	if run.length > 1 {
		if run.result == Win {
			h.removeWinLen(run.length)
			h.runs[front].length = run.length - 1
			h.addWinLen(run.length - 1)
		} else {
			h.runs[front].length = run.length - 1
		}
		return
	}
	if run.result == Win {
		h.removeWinLen(1)
	}
	h.runs[front] = winRun{}
	h.runsLen--
}

// Record 记一场对局，按调用顺序追加；全场与窗口统计就地更新，不保留全量记录。
func (h *History) Record(match Match) {
	result := match.Result

	h.played++
	switch result {
	case Win:
		h.wins++
		h.currentStreak++
		if h.currentStreak > h.maxStreak {
			h.maxStreak = h.currentStreak
		}
	case Draw:
		h.draws++
		h.currentStreak = 0
	case Loss:
		h.losses++
		h.currentStreak = 0
	}

	if h.ringFull {
		h.evictWindow(h.ring[h.ringPos])
	}
	h.ring[h.ringPos] = result
	h.ringPos = (h.ringPos + 1) % RecentWindow
	if h.played >= RecentWindow {
		h.ringFull = true
	}

	switch result {
	case Win:
		h.wWins++
	case Draw:
		h.wDraws++
	case Loss:
		h.wLosses++
	}

	if h.runsLen == 0 {
		h.appendRun(result)
	} else {
		last := (h.runsPos - 1 + RecentWindow) % RecentWindow
		if h.runs[last].result == result {
			if result == Win {
				h.growLastRun()
			} else {
				h.runs[last].length++
			}
		} else {
			h.appendRun(result)
		}
	}
}

// evictWindow 处理一场旧对局离开最近窗口。
func (h *History) evictWindow(old Result) {
	switch old {
	case Win:
		h.wWins--
	case Draw:
		h.wDraws--
	case Loss:
		h.wLosses--
	}
	h.evictFront()
}

// Summary 是全场统计，直接返回在线累积的结果，不遍历任何历史记录。
func (h *History) Summary() Summary {
	return Summary{
		Played:        h.played,
		Wins:          h.wins,
		Draws:         h.draws,
		Losses:        h.losses,
		WinRate:       winRate(h.wins, h.played),
		MaxStreak:     h.maxStreak,
		CurrentStreak: h.currentStreak,
	}
}

// Recent 是最近 RecentWindow 场的统计（不足就是全部），直接读在线维护的窗口状态。
func (h *History) Recent() Summary {
	played := h.wWins + h.wDraws + h.wLosses
	current := 0
	if h.runsLen > 0 {
		last := (h.runsPos - 1 + RecentWindow) % RecentWindow
		if h.runs[last].result == Win {
			current = h.runs[last].length
		}
	}
	return Summary{
		Played:        played,
		Wins:          h.wWins,
		Draws:         h.wDraws,
		Losses:        h.wLosses,
		WinRate:       winRate(h.wWins, played),
		MaxStreak:     h.winMaxLen,
		CurrentStreak: current,
	}
}

// Scanned 是查询接口累计扫过的记录数（规模观测）。
// 查询为 O(1) 只读，不扫历史记录，因此恒为 0。
func (h *History) Scanned() int { return 0 }

// Len 是已记录的对局数。
func (h *History) Len() int { return h.played }

func winRate(wins, played int) float64 {
	if played == 0 {
		return 0
	}
	return round1(float64(wins) * 100 / float64(played))
}

func round1(value float64) float64 {
	return float64(int64(value*10+0.5)) / 10
}
