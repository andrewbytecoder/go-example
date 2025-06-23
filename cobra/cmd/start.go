package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动服务",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("正在启动服务...")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
