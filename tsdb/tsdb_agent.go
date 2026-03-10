package tsdb

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/prometheus/prometheus/util/stats"
)

func main() {
	// 1. 指定 TSDB 数据目录路径
	// 这通常是 Prometheus 启动时 --storage.tsdb.path 指定的目录
	dbPath := "./data"

	// 2. 以只读模式打开 DB
	// 注意：如果 Prometheus Server 正在运行并占用此目录，此操作会失败或导致严重问题
	logger := log.New(&logWriter{}, "", 0) // 简单日志适配器

	opts := &tsdb.Options{
		RetentionDuration:    0,    // 只读模式下不重要
		NoLockfile:           true, // 跳过锁文件检查 (危险，确保没有其他进程在使用)
		WALReplayConcurrency: 1,
	}

	// 打开数据库 (ReadOnly)
	// 注意：不同版本的 tsdb.Open API 可能有所不同，此处基于较新版本
	db, err := tsdb.Open(dbPath, logger, nil, opts, stats.NewQueryStats(nil))
	if err != nil {
		log.Fatalf("Failed to open TSDB: %v", err)
	}
	defer db.Close()

	// 3. 创建一个 Querier
	// 时间范围：最近 1 小时
	mint := time.Now().Add(-1 * time.Hour).UnixMilli()
	maxt := time.Now().UnixMilli()

	querier, err := db.Querier(mint, maxt)
	if err != nil {
		log.Fatalf("Failed to create querier: %v", err)
	}
	defer querier.Close()

	// 4. 定义查询选择器 (Selector)
	// 相当于 PromQL: node_cpu_seconds_total{mode="idle"}
	matchers := []*labels.Matcher{
		labels.MustNewMatcher(labels.MatchEqual, "__name__", "node_cpu_seconds_total"),
		labels.MustNewMatcher(labels.MatchEqual, "mode", "idle"),
	}

	// 5. 执行查询
	seriesSet := querier.Select(context.TODO(), false, nil, matchers...)

	// 6. 遍历结果
	count := 0
	for seriesSet.Next() {
		series := seriesSet.At()
		labels := series.Labels()

		fmt.Printf("Series Labels: %s\n", labels.String())

		// 遍历该时间序列的所有样本点
		it := series.Iterator(nil)
		pointCount := 0
		for it.Next() == storage.ValFloat {
			ts, val := it.At()
			if pointCount < 5 { // 仅打印前 5 个点以免刷屏
				fmt.Printf("  Time: %v, Value: %f\n", time.UnixMilli(ts), val)
			}
			pointCount++
		}
		if err := it.Err(); err != nil {
			log.Printf("Iterator error: %v", err)
		}
		fmt.Printf("  Total points in range: %d\n\n", pointCount)

		count++
		if count >= 3 { // 限制只显示 3 个序列
			break
		}
	}

	if err := seriesSet.Err(); err != nil {
		log.Fatalf("Select error: %v", err)
	}
}

// 简单的日志适配器，满足 tsdb.Logger 接口
type logWriter struct{}

func (l *logWriter) Write(p []byte) (n int, err error) {
	fmt.Print(string(p))
	return len(p), nil
}
