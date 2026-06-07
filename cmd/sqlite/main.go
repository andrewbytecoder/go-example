package main

import (
	"fmt"
	"log"

	"github.com/glebarez/sqlite" // 纯 Go 实现的 SQLite 驱动，无需 CGO
	"gorm.io/gorm"
)

// 定义一个测试结构体
type User struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:100"`
	Age  int
}

func (u *User) TableName() string {
	return "users"
}

func main() {
	// 1. 使用纯 Go 驱动打开 SQLite 数据库
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	// 2. 开启 mmap 内存映射 (例如设置 256MB)
	// 注意：PRAGMA 是基于当前连接（Session）生效的
	if err := db.Exec("PRAGMA mmap_size = 268435456").Error; err != nil {
		log.Fatal("设置 mmap_size 失败:", err)
	}

	// 3. 强烈建议：限制最大打开连接数为 1
	// 如果不限制，连接池中的其他连接可能无法享受到 mmap 配置
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("获取底层 sql.DB 失败:", err)
	}
	sqlDB.SetMaxOpenConns(1)

	// 4. 自动迁移（建表）
	if err := db.AutoMigrate(&User{}); err != nil {
		log.Fatal("自动迁移失败:", err)
	}

	// 5. 保存一个结构体
	user := User{Name: "Alice", Age: 25}
	if err := db.Save(&user).Error; err != nil {
		log.Fatal("保存数据失败:", err)
	}

	fmt.Printf("成功保存用户: ID=%d, Name=%s, Age=%d\n", user.ID, user.Name, user.Age)
}
