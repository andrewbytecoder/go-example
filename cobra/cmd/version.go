package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var name string

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("MyApp v1.0.0")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().StringVarP(&name, "name", "n", "", "name of the person to greet")
}
