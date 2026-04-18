package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/taito-project/taito/internal/cache"
	"github.com/taito-project/taito/ui"
)

var dryRunFlag bool

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove all cached packages",
	Long: `Removes all cached artifacts from the taito packages cache directory
(e.g. ~/.cache/taito/packages/).

Use --dry-run to see what would be removed without actually deleting anything.

Examples:
  taito prune              # removes everything in cache
  taito prune --dry-run    # shows what would be removed`,
	Run: func(cmd *cobra.Command, args []string) {
		cacheDir, err := cfgFromCmd(cmd).CacheDir()
		if err != nil {
			fmt.Printf(" %s  Failed to prune cache\n    %v\n", ui.FailIcon(), err)
			os.Exit(1)
		}

		packagesDir := filepath.Join(cacheDir, "packages")
		idx, _ := cache.LoadIndex(cacheDir)

		entries, err := cache.ScanPackages(packagesDir)
		if err != nil {
			fmt.Printf(" %s  Failed to prune cache\n    %v\n", ui.FailIcon(), err)
			os.Exit(1)
		}

		if len(entries) == 0 {
			fmt.Printf(" %s  Cache is already empty\n", ui.SuccessIcon())
			return
		}

		var totalSize int64
		for _, e := range entries {
			totalSize += e.Size
		}

		if dryRunFlag {
			fmt.Printf(" %s  Dry run — %d %s would be removed (%s)\n",
				ui.SuccessIcon(), len(entries), cache.ItemWord(len(entries)), cache.FormatSize(totalSize))
			for _, e := range entries {
				fmt.Printf("    - %s  (%s)\n", cache.EntryDisplayName(e, idx), cache.FormatSize(e.Size))
			}
			return
		}

		if err := cache.RemovePackages(packagesDir, entries); err != nil {
			fmt.Printf(" %s  Failed to prune cache\n    %v\n", ui.FailIcon(), err)
			os.Exit(1)
		}

		_ = cache.ClearIndex(cacheDir)

		fmt.Printf(" %s  Removed %d %s (%s)\n",
			ui.SuccessIcon(), len(entries), cache.ItemWord(len(entries)), cache.FormatSize(totalSize))
		for _, e := range entries {
			fmt.Printf("    - %s  (%s)\n", cache.EntryDisplayName(e, idx), cache.FormatSize(e.Size))
		}
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)

	pruneCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Show what would be removed without deleting")
}
