// Command match-history 跑战绩统计样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"example.com/matchhistory"
)

func record(history *matchhistory.History, index int, result matchhistory.Result) {
	history.Record(matchhistory.Match{ID: fmt.Sprintf("m-%d", index), AtMs: int64(index) * 1000, Result: result})
}

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("match-history", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "draw", "draw / streak / recent / cost")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	history := matchhistory.NewHistory()
	switch *sample {
	case "draw":
		record(history, 0, matchhistory.Win)
		record(history, 1, matchhistory.Draw)
		record(history, 2, matchhistory.Loss)
		summary := history.Summary()
		fmt.Fprintf(stdout, "wins=%d draws=%d losses=%d winrate=%.1f maxstreak=%d currentstreak=%d\n",
			summary.Wins, summary.Draws, summary.Losses, summary.WinRate, summary.MaxStreak, summary.CurrentStreak)
	case "streak":
		for index := 0; index < 3; index++ {
			record(history, index, matchhistory.Win)
		}
		record(history, 3, matchhistory.Draw)
		record(history, 4, matchhistory.Win)
		summary := history.Summary()
		fmt.Fprintf(stdout, "maxstreak=%d currentstreak=%d\n", summary.MaxStreak, summary.CurrentStreak)
	case "recent":
		// 前 20 场全负，后 100 场全胜。
		for index := 0; index < 20; index++ {
			record(history, index, matchhistory.Loss)
		}
		for index := 20; index < 120; index++ {
			record(history, index, matchhistory.Win)
		}
		recent := history.Recent()
		fmt.Fprintf(stdout, "played=%d wins=%d losses=%d winrate=%.1f maxstreak=%d\n",
			recent.Played, recent.Wins, recent.Losses, recent.WinRate, recent.MaxStreak)
	case "cost":
		// 二十万场之后连续查一万次滚动窗口。
		for index := 0; index < 200000; index++ {
			result := matchhistory.Win
			if index%3 == 0 {
				result = matchhistory.Loss
			}
			record(history, index, result)
		}
		for query := 0; query < 10000; query++ {
			history.Recent()
		}
		fmt.Fprintf(stdout, "scanned=%d\n", history.Scanned())
	default:
		fmt.Fprintln(stderr, "需要 --sample draw|streak|recent|cost")
		return 2
	}
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
