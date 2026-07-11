package pidfile

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// CreatePidFile 将当前进程的 PID 及相关信息写入指定文件
// 如果文件已存在且被其他进程锁定，会返回错误，防止多实例运行
func CreatePidFile(pidFile string) error {
	// 1. 确保目录存在
	dir := filepath.Dir(pidFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建pid文件目录失败: %w", err)
	}

	// 2. 打开或创建文件
	f, err := os.OpenFile(pidFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("打开pid文件失败: %w", err)
	}
	defer f.Close()

	// 3. 尝试获取文件锁（非阻塞），防止多实例运行
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("获取文件锁失败，可能已有其他实例在运行: %w", err)
	}

	// 4. 收集进程相关信息
	pid := os.Getpid()
	startTime := time.Now().Format(time.RFC3339)
	execPath, _ := os.Executable()
	hostname, _ := os.Hostname()

	// 5. 格式化并写入内容
	content := fmt.Sprintf("PID: %d\nStartTime: %s\nExecutable: %s\nHostname: %s\n",
		pid, startTime, execPath, hostname)

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("写入pid文件失败: %w", err)
	}

	// 6. 强制同步到磁盘
	if err := f.Sync(); err != nil {
		return fmt.Errorf("同步pid文件到磁盘失败: %w", err)
	}

	return nil
}

func RemovePidFile(pidFile string) {
	// 打开文件以释放锁
	if f, err := os.Open(pidFile); err == nil {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}
	os.Remove(pidFile)
}
