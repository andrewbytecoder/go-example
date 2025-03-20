package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// 1. 调用父Cancel能关闭子 context
// 2. 调用子Cancel不能关闭父 context

func main() {
	// 创建父 Context
	parentCtx, pCancel := context.WithCancel(context.Background())
	defer pCancel()

	// 创建子 Context
	childCtx, cancel := context.WithCancel(parentCtx)

	// 模拟子 Context 调用 Cancel
	cancel() // 取消子 Context

	// 检查父 Context 的状态
	select {
	case <-parentCtx.Done():
		fmt.Println("父 Context 已被取消")
	case <-time.After(1 * time.Second):
		fmt.Println("父 Context 未被取消") // 输出此结果
	}

	// 检查父 Context 的状态
	select {
	case <-childCtx.Done():
		fmt.Println("子 Context 已被取消")
	case <-time.After(1 * time.Second):
		fmt.Println("子 Context 未被取消") // 输出此结果
	}

	// 检查子 Context 的状态
	if !errors.Is(childCtx.Err(), context.Canceled) {
		fmt.Println("子 Context 未被取消") // 不会执行
	} else {
		fmt.Println("子 Context 已被取消") // 执行此分支
	}
}
