package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止服务",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("正在停止服务...")
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
