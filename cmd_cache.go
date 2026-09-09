package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// cacheCmd — önbelleği görüntüle / temizle.
var cacheCmd = &cobra.Command{
	Use:   "cache [list|clear]",
	Short: "Show or clear the answer cache",
	Long: `  cem cache          → what is stored, and how old
  cem cache clear    → delete everything

  The thinker's answers are cached, so asking the same thing twice does not
  pay for the same reasoning twice. The writer is NOT cached by default: it
  creates files, and replaying its answer would leave none behind.
  Enable it with cache_writer: true in ~/.cem/config.yaml.
  Skip the cache for a single run with --no-cache.`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		entries, paths, err := loadCacheEntries()
		if err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}

		if len(args) == 1 && args[0] == "clear" {
			for _, p := range paths {
				_ = os.Remove(p)
			}
			fmt.Printf("  %s %s\n", styleSuccess.Render("✓"),
				fmt.Sprintf(L("%d kayıt silindi", "%d entries deleted"), len(paths)))
			return
		}

		if len(entries) == 0 {
			fmt.Println(styleDim.Render(L("  Önbellek boş.", "  The cache is empty.")))
			return
		}

		fmt.Println()
		fmt.Println(styleBold.Render(fmt.Sprintf(
			L("  Önbellek — %d kayıt", "  Cache — %d entries"), len(entries))))
		fmt.Println()
		for i, e := range entries {
			if i >= 20 {
				fmt.Println(styleDim.Render(fmt.Sprintf(
					L("  … ve %d kayıt daha", "  … and %d more"), len(entries)-20)))
				break
			}
			desc := e.Tool
			if e.Model != "" {
				desc += " · " + e.Model
			}
			if e.Effort != "" {
				desc += " · " + e.Effort
			}
			fmt.Printf("  %s %-9s %s\n",
				styleDim.Render(formatDuration(nowSince(e.Created))+" "+L("önce", "ago")),
				e.Role, styleDim.Render(desc))
			fmt.Printf("      %s\n", styleBold.Render(firstLine(e.Input, 70)))
		}
		fmt.Println()
		fmt.Println(styleDim.Render(L("  cem cache clear   → hepsini sil",
			"  cem cache clear   → delete everything")))
		fmt.Println(styleDim.Render(L("  cem --no-cache …  → bu çalıştırmada önbelleği atla",
			"  cem --no-cache …  → skip the cache for one run")))
		fmt.Println()
	},
}

// firstLine — çok satırlı promptun ilk satırı, kırpılmış.
func firstLine(s string, max int) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	r := []rune(s)
	if len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}

func init() {
	rootCmd.AddCommand(cacheCmd)
}
