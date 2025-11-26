package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"size:255"`
	Email string `gorm:"size:255"`
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pgsql",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of your application. For example:

my-cli server --port=8080`,
}

var cfgFile string
var host string
var port string
var user string
var password string

func initConfig() {
	// 这里可以加载配置文件（如 viper）
	if cfgFile != "" {
		// 使用 cfgFile 路径加载配置
		fmt.Printf("Using config file: %s\n", cfgFile)
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func init() {
	cobra.OnInitialize(initConfig)

	// 全局标志：--config
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.my-cli.yaml)")
	rootCmd.PersistentFlags().StringVarP(&user, "user", "u", "admin", "user name")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "Bingo@1993", "password")
	rootCmd.PersistentFlags().StringVarP(&host, "host", "a", "10.206.114.9", "password")
	rootCmd.PersistentFlags().StringVarP(&port, "port", "d", "3433", "password")
}

func Start() {

	// 安全处理含特殊字符的密码并添加超时设置
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=disable TimeZone=Asia/Shanghai connect_timeout=10",
		host, user, password, port)
	fmt.Println("🚀 连接数据库...", dsn)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ 连接失败:", err)
	} else {
		fmt.Println("🚀 连接成功")
	}

	// 只进行查询操作
	var user User
	result := db.First(&user)
	if result.Error != nil {
		fmt.Printf("未找到用户数据: %v\n", result.Error)
	} else {
		fmt.Printf("👤 查询到用户: %+v\n", user)
	}

	// 原生 SQL 查询
	var count int64
	db.Raw("SELECT COUNT(*) FROM users").Scan(&count)
	fmt.Printf("🔢 用户总数: %d\n", count)

}

func main() {
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		Start()
	}
	Execute()
}
