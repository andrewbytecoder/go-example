package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq" // PostgreSQL driver
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
var dbName string

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
	rootCmd.PersistentFlags().StringVarP(&dbName, "db", "b", "postgres", "database name")
}

type UserDatabase struct {
	Name string `gorm:"column:name"`
}

func createDatabaseIfNotExists(host, user, password string, port string) error {
	// 1. 先连接到默认的 'postgres' 数据库（通常总是存在）
	defaultDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=disable",
		host, user, password, port)
	fmt.Println("🚀 连接数据库...", defaultDSN)
	db, err := gorm.Open(postgres.Open(defaultDSN), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ 连接失败:", err)
		return fmt.Errorf("failed to connect to default 'postgres' db: %w", err)
	} else {
		fmt.Println("🚀 默认数据库连接成功")
	}

	// 2. 检查目标数据库是否存在
	var exists bool
	err = db.Raw("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = ?)", dbName).Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}

	// 3. 如果不存在，则创建
	if !exists {
		fmt.Printf("Database '%s' does not exist, creating...\n", dbName)
		err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(dbName))).Error
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		fmt.Printf("✅ Database '%s' created successfully.\n", dbName)
	}
	var userDatabases []UserDatabase

	// 查询用户数据库（排除系统数据库）
	err = db.Raw(`
		SELECT datname as name 
		FROM pg_database 
		WHERE datistemplate = false 
		AND datallowconn = true
		AND datname NOT IN ('postgres', 'template0', 'template1')
		ORDER BY datname
	`).Scan(&userDatabases).Error

	if err != nil {
		log.Fatal("查询用户数据库失败:", err)
	}

	fmt.Printf("当前 PostgreSQL 实例共有 %d 个用户数据库:\n", len(userDatabases))
	fmt.Println("==============================================")

	if len(userDatabases) == 0 {
		fmt.Println("未找到用户数据库")
		return fmt.Errorf("未找到用户数据库")
	}

	for i, dbInfo := range userDatabases {
		fmt.Printf("%d. %s\n", i+1, dbInfo.Name)
	}

	// 只返回数量
	fmt.Printf("\n✅ 用户数据库总数: %d\n", len(userDatabases))

	return nil
}

// 防 SQL 注入：对标识符加引号（PostgreSQL 使用双引号）
func quoteIdentifier(s string) string {
	return "\"" + s + "\""
}
func main() {
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		err := createDatabaseIfNotExists(host, user, password, port)
		if err != nil {
			log.Fatal("创建数据库失败:", err)
			return
		}
	}
	Execute()
}
