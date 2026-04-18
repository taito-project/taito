package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of taito",
	Long:  `All software has versions. This is taito's.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("taito version " + Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
