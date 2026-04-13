package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	// 1. 解析服务器地址
	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:9999")
	if err != nil {
		fmt.Println("Error resolving server address:", err)
		os.Exit(1)
	}

	// 2. 创建本地连接
	// DialUDP 的第二个参数 (LocalAddr) 设为 nil，表示让系统自动分配本地端口
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		fmt.Println("Error dialing:", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("UDP Client connected to server.")

	// 3. 从标准输入读取用户消息并发送
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter message to send (type 'exit' to quit): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "exit" {
			break
		}

		// 发送数据
		_, err := conn.Write([]byte(input))
		if err != nil {
			fmt.Println("Error sending:", err)
			break
		}

		// 4. 接收回复
		buffer := make([]byte, 1024)
		// 因为使用了 DialUDP，可以直接 Read，不需要指定来源地址
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error receiving:", err)
			break
		}

		fmt.Printf("Received from server: %s\n", string(buffer[:n]))
	}
}
