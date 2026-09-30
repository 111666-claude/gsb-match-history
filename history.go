// Package matchhistory 是战绩统计：全场统计随记录在线累积，最近 RecentWindow 场用滚动窗口维护。
// Record 之后 Summary 与 Recent 都是就地更新后的 O(1) 结果，内存里只留最近 RecentWindow 场。
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
//
// 全场统计用普通计数器在线维护；最近 RecentWindow 场存进定长环形缓冲，
// 窗口里的最长连胜按“非胜屏障之间的胜场段”维护：缓冲满后每场淘汰最早一场，
// 被关闭的内部胜场段进单调队列，开口的前/后缀段随窗口边界直接算，Record 摊还 O(1)。
type History struct {
	// 全场统计。
	played        int
	wins          int
	draws         int
	losses        int
	maxStreak     int
	currentStreak int
	scanned       int
	// 滚动窗口：只保留最近 RecentWindow 场的结果。
	ring         [RecentWindow]Result
	ringHead     int
	windowFilled bool
	windowWins   int
	windowDraws  int
	windowLosses int
	suffixStreak int
	// 屏障队列：窗口内按时间顺序的非胜（平局/败局）全局序号，定长环形队列。
	barriers     [RecentWindow]int64
	barrierHead  int
	barrierCount int
	// 单调队列：已关闭的内部胜场段（长度递减），记录段长与其到期屏障序号。
	segLengths [RecentWindow]int
	segExpiry  [RecentWindow]int64
	segHead    int
	segCount   int
}

// NewHistory 建一个空的历史。
func NewHistory() *History {
	return &History{}
}

// Record 记一场对局，按调用顺序追加，全场统计与滚动窗口都就地更新。
func (h *History) Record(match Match) {
	// 全场统计。
	h.played++
	switch match.Result {
	case Win:
		h.wins++
		h.currentStreak++
		if h.currentStreak > h.maxStreak {
			h.maxStreak = h.currentStreak
		}
	case Draw:
		h.draws++
		h.currentStreak = 0
	default:
		h.losses++
		h.currentStreak = 0
	}

	// 滚动窗口：缓冲已满时先淘汰最早一场。
	if h.windowFilled {
		h.evict(int64(h.played - 1 - RecentWindow))
	}
	slot := h.played - 1
	if h.windowFilled {
		slot = h.ringHead
	}
	h.ring[slot] = match.Result

	switch match.Result {
	case Win:
		h.windowWins++
		h.suffixStreak++
	default:
		if match.Result == Draw {
			h.windowDraws++
		} else {
			h.windowLosses++
		}
		h.closeRun(int64(h.played - 1))
		h.suffixStreak = 0
	}

	if h.played < RecentWindow {
		// 还没填满，head 指向下一空位。
		h.ringHead = h.played
	} else {
		// 已满，新场复用了最早场的槽位，head 推进到下一场（仍是最早一场）。
		h.windowFilled = true
		h.ringHead = h.played % RecentWindow
	}
}

// closeRun 在窗口尾部落一个非胜屏障 barrierIndex 时，关闭它之前的后缀胜场段。
// 队列里没有更早屏障时，这段是开口前缀，不进单调队列（前缀长度随淘汰直接算）。
func (h *History) closeRun(barrierIndex int64) {
	length := h.suffixStreak
	h.pushBarrier(barrierIndex)
	if length == 0 || h.barrierCount < 2 {
		return
	}
	previous := h.barrierAt(h.barrierCount - 2)
	for h.segCount > 0 {
		tail := (h.segHead + h.segCount - 1) % RecentWindow
		if h.segLengths[tail] > length {
			break
		}
		h.segCount--
	}
	index := (h.segHead + h.segCount) % RecentWindow
	h.segLengths[index] = length
	h.segExpiry[index] = previous
	h.segCount++
}

// evict 从窗口淘汰最早一场（全局序号 evictedIndex）。
func (h *History) evict(evictedIndex int64) {
	switch h.ring[h.ringHead] {
	case Win:
		h.windowWins--
		// 窗口里没有非胜屏障时，整窗连胜就是后缀段，淘汰胜场要同步缩短。
		if h.barrierCount == 0 {
			h.suffixStreak--
		}
	case Draw:
		h.windowDraws--
		h.dropBarrier(evictedIndex)
	default:
		h.windowLosses--
		h.dropBarrier(evictedIndex)
	}
}

// dropBarrier 淘汰最早屏障 barrierIndex：它关闭的内部段到期出队。
func (h *History) dropBarrier(barrierIndex int64) {
	if h.segCount > 0 && h.segExpiry[h.segHead] == barrierIndex {
		h.segHead = (h.segHead + 1) % RecentWindow
		h.segCount--
	}
	h.barrierHead = (h.barrierHead + 1) % RecentWindow
	h.barrierCount--
}

// pushBarrier 把一个非胜屏障序号追加到屏障队列尾部。
func (h *History) pushBarrier(index int64) {
	slot := (h.barrierHead + h.barrierCount) % RecentWindow
	h.barriers[slot] = index
	h.barrierCount++
}

// barrierAt 返回屏障队列里第 position 个（0 起）屏障的全局序号。
func (h *History) barrierAt(position int) int64 {
	return h.barriers[(h.barrierHead+position)%RecentWindow]
}

// windowMaxStreak 是当前窗口内的最长连胜，O(1)：
// 后缀开口段（suffixStreak）、内部已关闭段（单调队列队首）、前缀开口段（首屏障之前）取最大。
func (h *History) windowMaxStreak() int {
	best := h.suffixStreak
	if h.barrierCount == 0 {
		// 整窗全胜（含空窗），所有场次都在开口段里。
		if h.windowPlayed() > best {
			best = h.windowPlayed()
		}
		return best
	}
	first := h.barrierAt(0)
	left := int64(h.played) - int64(h.windowPlayed())
	if prefix := int(first - left); prefix > best {
		best = prefix
	}
	if h.segCount > 0 && h.segLengths[h.segHead] > best {
		best = h.segLengths[h.segHead]
	}
	return best
}

// windowPlayed 是当前窗口里的场次数。
func (h *History) windowPlayed() int {
	if h.windowFilled {
		return RecentWindow
	}
	return h.played
}

// Summary 是全场统计，直接返回在线维护的结果，不遍历任何记录。
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

// Recent 是最近 RecentWindow 场的统计（不足 RecentWindow 场时就是全部），直接返回在线维护的结果。
func (h *History) Recent() Summary {
	played := h.windowPlayed()
	return Summary{
		Played:        played,
		Wins:          h.windowWins,
		Draws:         h.windowDraws,
		Losses:        h.windowLosses,
		WinRate:       winRate(h.windowWins, played),
		MaxStreak:     h.windowMaxStreak(),
		CurrentStreak: h.suffixStreak,
	}
}

// Scanned 是查询接口累计扫过的记录数（规模观测）。
// Summary 与 Recent 都是 O(1) 摊还，这个计数不随查询次数增长。
func (h *History) Scanned() int { return h.scanned }

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
