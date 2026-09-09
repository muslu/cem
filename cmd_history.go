package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	historyLimit int
	historyClear bool
)

func init() {
	historyCmd.Flags().IntVarP(&historyLimit, "limit", "n", 20, "show the last N entries")
	historyCmd.Flags().BoolVar(&historyClear, "clear", false, "clear history.log")
	rootCmd.AddCommand(historyCmd)
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show command history",
	Long: `  cem history          → son 20 satır
  cem history -n 100   → son 100 satır
  cem history --clear  → log'u temizle

  Log: ~/.cem/history.log (TSV: timestamp, mode, role, exit, input)`,
	Run: func(cmd *cobra.Command, args []string) {
		path, err := historyPath()
		if err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}

		if historyClear {
			if !askYN("  history.log silinsin mi?") {
				fmt.Println(styleDim.Render(L("  İptal.", "  Cancelled.")))
				return
			}
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				fmt.Println(styleError.Render("✗ " + err.Error()))
				os.Exit(1)
			}
			fmt.Println(styleSuccess.Render("  ✓ Temizlendi"))
			return
		}

		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println(styleDim.Render(L("  Henüz geçmiş yok.", "  No history yet.")))
				return
			}
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}
		defer f.Close()

		// Son N satırı tut
		var lines []string
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			lines = append(lines, sc.Text())
			if len(lines) > historyLimit*4 { // ring boyutunu sınırla
				lines = lines[len(lines)-historyLimit:]
			}
		}
		if err := sc.Err(); err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}
		if len(lines) > historyLimit {
			lines = lines[len(lines)-historyLimit:]
		}

		if len(lines) == 0 {
			fmt.Println(styleDim.Render(L("  Henüz geçmiş yok.", "  No history yet.")))
			return
		}

		fmt.Println(styleBold.Render(fmt.Sprintf(L("  Son %d kayıt", "  Last %d entries"), len(lines))))
		fmt.Println(styleDim.Render(strings.Repeat("─", 70)))
		for _, line := range lines {
			parts := strings.SplitN(line, "\t", 5)
			if len(parts) < 5 {
				continue
			}
			ts, mode, role, exitStr, input := parts[0], parts[1], parts[2], parts[3], parts[4]

			when := ts
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				when = t.Local().Format("01-02 15:04")
			}

			marker := styleSuccess.Render("✓")
			if code, _ := strconv.Atoi(exitStr); code != 0 {
				marker = styleError.Render("✗")
			}

			fmt.Printf("  %s %s  %s  %s  %s\n",
				marker,
				styleDim.Render(when),
				styleBold.Render(fmt.Sprintf("%-5s", mode)),
				styleDim.Render(fmt.Sprintf("%-15s", role)),
				input,
			)
		}
		fmt.Println()
		fmt.Println(styleDim.Render("  Log: ~/.cem/history.log"))
	},
}
