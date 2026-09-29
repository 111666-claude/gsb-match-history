// Command match-history 跑战绩统计样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"example.com/matchhistory"
)

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("match-history", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "draw", "draw / streak / unsorted")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	var matches []matchhistory.Match
	switch *sample {
	case "draw":
		matches = []matchhistory.Match{
			{ID: "w1", AtMs: 1000, Result: matchhistory.Win},
			{ID: "d1", AtMs: 2000, Result: matchhistory.Draw},
			{ID: "l1", AtMs: 3000, Result: matchhistory.Loss},
		}
	case "streak":
		matches = []matchhistory.Match{
			{ID: "w1", AtMs: 1000, Result: matchhistory.Win},
			{ID: "w2", AtMs: 2000, Result: matchhistory.Win},
			{ID: "w3", AtMs: 3000, Result: matchhistory.Win},
			{ID: "d1", AtMs: 4000, Result: matchhistory.Draw},
			{ID: "w4", AtMs: 5000, Result: matchhistory.Win},
		}
	case "unsorted":
		matches = []matchhistory.Match{
			{ID: "l1", AtMs: 3000, Result: matchhistory.Loss},
			{ID: "w1", AtMs: 1000, Result: matchhistory.Win},
			{ID: "d1", AtMs: 2000, Result: matchhistory.Draw},
		}
	default:
		fmt.Fprintln(stderr, "需要 --sample draw|streak|unsorted")
		return 2
	}
	summary := matchhistory.Summarize(matches)
	fmt.Fprintf(stdout, "played=%d wins=%d draws=%d losses=%d winrate=%.1f maxstreak=%d currentstreak=%d\n",
		summary.Played, summary.Wins, summary.Draws, summary.Losses,
		summary.WinRate, summary.MaxStreak, summary.CurrentStreak)
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
