package main

import (
	"flag"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"k8s.io/klog/v2"
)

func main() {
	// 1. 初始化你的 Zap Logger
	// 生产环境建议使用 NewProductionConfig()，开发环境用 NewDevelopmentConfig()
	config := zap.NewDevelopmentConfig()
	// 可选：自定义编码格式，使其更像 klog 或 JSON
	config.Encoding = "json"
	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	// 2. 创建 klog 的适配器
	zapSink := &ZapLogSink{
		logger: logger,
	}

	// 2. 显式设置默认级别或解析命令行参数
	// 方式 A: 硬编码设置级别为 2 (开发调试常用)
	// 3. 【关键步骤】将 klog 的全局 Logger 替换为我们的 Zap 适配器
	// 注意：klog 是全局单例，这会影响整个进程中所有使用 klog 的库
	klog.SetLogger(logr.New(zapSink))

	// 4. (可选) 设置 klog 的 verbosity 级别
	// 如果你想看到 client-go 的详细调试信息 (如 HTTP 请求)，需要设置较高的 V 值
	// 可以通过 flag 解析，或者直接硬编码测试
	klog.InitFlags(nil) // 初始化 klog 的 flag 解析
	// 模拟设置 -v=2 (Debug 级别)
	// 在实际生产中，建议通过命令行参数 --v 来控制
	flag.Set("v", "2")
	flag.Parse()

	// --- 现在可以安全地使用 client-go 了 ---

	// 示例：创建一个简单的配置 (这里只是演示，实际需加载 kubeconfig)
	// 当 client-go 内部打印日志时，输出将会经过上面的 ZapLogSink，最终由 Zap 输出
	/*
		config, _ := rest.InClusterConfig()
		clientset, _ := kubernetes.NewForConfig(config)
		_, _ = clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})
	*/

	logger.Info("Klog has been redirected to Zap successfully!")

	// 模拟一条 klog 输出来验证
	klog.V(1).InfoS("This is a test message from klog", "component", "test-adapter")
}
