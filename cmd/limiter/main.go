package main

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"
)

func main() {
	// 每秒 5 个 token，桶容量 5
	limiter := rate.NewLimiter(5, 5)

	for i := 0; i < 20; i++ {
		ctx := context.Background()
		err := limiter.Wait(ctx) // 阻塞直到拿到 token
		if err != nil {
			fmt.Println("err:", err)
			continue
		}
		fmt.Printf("request %d at %v\n", i, time.Now().Format("15:04:05.000"))
	}
}
