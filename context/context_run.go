package context

import (
	"context"
	"time"
)

func NewContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

func Run(ctx context.Context, ch chan<- int) {
	ticker := time.NewTicker(1 * time.Second)
	// 这里是Run 函数的栈，如果在这里调用defer就会导致Run退出defer函数被调用
	go func() {
		// go func 之后是一个函数中新的栈，只有这个函数退出的时候才会触发defer
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-ticker.C:
				// 模拟处理任务
				i++
				ch <- i
			case <-ctx.Done():
				// 监听到取消信号，结束协程
				return
			default:
			}
		}
	}()
}
