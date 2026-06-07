package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

const (
	writeCount = 100000 // 写入条数
	readCount  = 80000  // 随机读取次数
	blobSize   = 8192   // 每条记录负载 8KB → 总数据 ~800MB
)

// BigRecord 大结构体，模拟真实业务数据
type BigRecord struct {
	ID       uint   `gorm:"primaryKey"`
	Title    string `gorm:"size:256"`
	Content  string `gorm:"size:10000"` // 大数据字段
	Tag      string `gorm:"size:128"`
	Priority int
	Status   int
	Score    float64
}

// genBlob 生成指定大小的随机文本
func genBlob(size int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789  "
	var sb strings.Builder
	sb.Grow(size)
	for i := 0; i < size; i++ {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return sb.String()
}

// runFullBench 完整测试：先写入大量数据，再做随机读取
func runFullBench(t *testing.T, label, dsn, dbName string) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal("数据库连接失败:", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal("获取底层 sql.DB 失败:", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&BigRecord{}); err != nil {
		t.Fatal("自动迁移失败:", err)
	}

	// 预生成数据
	blobs := make([]string, writeCount)
	for i := 0; i < writeCount; i++ {
		blobs[i] = genBlob(blobSize)
	}

	fmt.Println()
	fmt.Printf("========== %s ==========\n", label)
	fmt.Printf("记录数: %d  |  单条负载: %d 字节  |  总数据量: ~%.1f MB\n",
		writeCount, blobSize, float64(writeCount*blobSize)/(1024*1024))

	// ---------- 写入阶段 ----------
	writeStart := time.Now()
	ids := make([]uint, 0, writeCount)
	for i := 0; i < writeCount; i++ {
		rec := BigRecord{
			Title:    fmt.Sprintf("Record_%d", i),
			Content:  blobs[i],
			Tag:      fmt.Sprintf("tag_%d", i%50),
			Priority: i % 10,
			Status:   i % 5,
			Score:    float64(i%100) * 0.95,
		}
		if err := db.Create(&rec).Error; err != nil {
			t.Fatal("写入失败:", err)
		}
		ids = append(ids, rec.ID)
	}
	writeElapsed := time.Since(writeStart)

	// ---------- 读取阶段（随机读） ----------
	rng := rand.New(rand.NewSource(42)) // 固定种子，公平对比
	readStart := time.Now()
	for i := 0; i < readCount; i++ {
		randIdx := rng.Intn(len(ids))
		var rec BigRecord
		if err := db.First(&rec, ids[randIdx]).Error; err != nil {
			t.Fatal("读取失败:", err)
		}
	}
	readElapsed := time.Since(readStart)

	// ---------- 汇总 ----------
	fmt.Println()
	fmt.Println("--- 写入阶段 ---")
	fmt.Printf("  总耗时:       %v\n", writeElapsed)
	fmt.Printf("  平均每次:     %.4f ms\n", writeElapsed.Seconds()*1000/float64(writeCount))
	fmt.Printf("  写入速度:     %.0f ops/s\n", float64(writeCount)/writeElapsed.Seconds())

	fmt.Println()
	fmt.Println("--- 随机读取阶段 ---")
	fmt.Printf("  总耗时:       %v\n", readElapsed)
	fmt.Printf("  平均每次:     %.4f ms\n", readElapsed.Seconds()*1000/float64(readCount))
	fmt.Printf("  读取速度:     %.0f ops/s\n", float64(readCount)/readElapsed.Seconds())

	fmt.Println()
	fmt.Println("--- 总计 ---")
	fmt.Printf("  总耗时:       %v\n", writeElapsed+readElapsed)
	fmt.Println("=======================================")
}

// 测试：使用 mmap
func TestSaveSpeedWithMmap(t *testing.T) {
	dbPath := "bench_mmap.db"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + "-wal")
	defer os.Remove(dbPath + "-shm")

	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=mmap_size(2147483648)", dbPath)
	runFullBench(t, "使用 mmap (2GB)", dsn, dbPath)
}

// 测试：不使用 mmap
func TestSaveSpeedNoMmap(t *testing.T) {
	dbPath := "bench_nommap.db"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + "-wal")
	defer os.Remove(dbPath + "-shm")

	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)", dbPath)
	runFullBench(t, "不使用 mmap", dsn, dbPath)
}

// 并排对比
func TestSaveSpeedCompare(t *testing.T) {
	t.Run("使用-mmap", TestSaveSpeedWithMmap)
	t.Run("不使用-mmap", TestSaveSpeedNoMmap)
}
