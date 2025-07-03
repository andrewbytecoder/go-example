package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cobra",
	Short: "My Cobra Application",
	Long:  `A longer description of my Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 当用户直接运行 ./cobra 或 ./cobra --help 时显示帮助
		cmd.Help()
		fmt.Println("rootCmd.Run")
		time.Sleep(60 * time.Second)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
