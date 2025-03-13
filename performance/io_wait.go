package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"
)

func highIOWaitSimulation(id int) {
	// 为每个goroutine创建一个独立的文件
	fileName := fmt.Sprintf("high_iowait_%d.txt", id)
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("Failed to create file %s: %v", fileName, err)
	}
	defer file.Close()

	fmt.Printf("Goroutine %d starting high iowait simulation...\n", id)

	// 持续写入数据
	for {
		// 写入 1MB 的数据
		data := make([]byte, 1024*1024*128) // 1MB
		_, err := file.Write(data)
		if err != nil {
			log.Fatalf("Goroutine %d failed to write to file: %v", id, err)
		}

		// 强制将数据写入磁盘
		err = file.Sync()
		if err != nil {
			log.Fatalf("Goroutine %d failed to sync file: %v", id, err)
		}

		// 打印日志
		fmt.Printf("Goroutine %d wrote 128MB of data to disk\n", id)

		// 等待一段时间
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	// 设置Go运行时可使用的最大CPU核心数
	runtime.GOMAXPROCS(6)

	fmt.Println("Starting high I/O wait simulation with 6 goroutines...")

	// 启动6个goroutine
	for i := 1; i <= 6; i++ {
		go highIOWaitSimulation(i)
	}

	// 防止主goroutine退出
	time.Sleep(time.Second * 100)
}
