package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "myapp",
	Short: "MyApp 是一个多命令服务管理工具",
	Long:  `MyApp 模拟 kubelet，支持 start, stop, status 等命令`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("欢迎使用 MyApp，输入 --help 查看可用命令")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
