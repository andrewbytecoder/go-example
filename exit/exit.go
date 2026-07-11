package exit

import (
	"log"
	"os"
	"os/signal"
)

// Exit codes. Generally, you should NOT
// automatically restart the process if the
// exit code is ExitCodeFailedStartup (1).
const (
	ExitCodeSuccess = iota
	ExitCodeFailedStartup
	ExitCodeForceQuit
	ExitCodeFailedQuit
)

// 程序能够优雅的退出，需要处理两种情况
// 1. 收到系统信号，如 SIGINT、SIGTERM 对内存等相关资源处理之后退出
// 2. 收到系统信号，如果短时间再次收到信号，则直接退出，这样可能会导致数据丢失

func TrapSignalsPlatform() {
	go func() {
		shutdown := make(chan os.Signal, 1)
		// 当接收到 SIGINT 的时候退出
		signal.Notify(shutdown, os.Interrupt)
		for i := 0; true; i++ {
			<-shutdown

			if i > 0 {
				log.Println("force quit", "Signal SIGINT")
				os.Exit(ExitCodeForceQuit)
			}

			log.Println("receive signal SIGINT")
			// 异步执行退出逻辑，防止阻塞，第二次接收到SIGINT时直接退出
			go exitProcessFromSignal()
		}
	}()
}

func exitProcessFromSignal() {
	// do something
	os.Exit(ExitCodeSuccess)
}
