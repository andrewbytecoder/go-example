package main

import (
	"context"
	"fmt"
	"log"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"

	"time"
)

func main() {
	ctx := context.Background()
	cli, _ := clientv3.New(clientv3.Config{Endpoints: []string{"localhost:2379"}, DialTimeout: 5 * time.Second})
	defer cli.Close()
	// 创建 Session，底层关联一个租约，默认会自动续期
	sess, _ := concurrency.NewSession(cli, concurrency.WithTTL(10))
	mutex := concurrency.NewMutex(sess, "/locks/task-1")
	// 加锁
	if err := mutex.Lock(ctx); err != nil {
		log.Fatal("获取锁失败:", err)
	}
	fmt.Println("成功获取 etcd 锁")
	// 模拟任务
	//processTask()
	//解锁
	if err := mutex.Unlock(ctx); err != nil {
		log.Fatal("释放锁失败:", err)
	}
}
