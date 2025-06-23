package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看服务状态",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("当前服务状态：运行正常")
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
