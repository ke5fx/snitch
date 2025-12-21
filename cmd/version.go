package cmd

import (
	"fmt"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version/build info",
	Run: func(cmd *cobra.Command, args []string) {
		bold := color.New(color.Bold)
		cyan := color.New(color.FgCyan)
		faint := color.New(color.Faint)

		bold.Print("snitch ")
		cyan.Println(Version)
		fmt.Println()

		faint.Print("  commit  ")
		fmt.Println(Commit)

		faint.Print("  built   ")
		fmt.Println(Date)

		faint.Print("  go      ")
		fmt.Println(runtime.Version())

		faint.Print("  os      ")
		fmt.Printf("%s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
