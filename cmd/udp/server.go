package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	// 1. 解析本地地址 (监听所有接口的 9999 端口)
	addr, err := net.ResolveUDPAddr("udp", ":9999")
	if err != nil {
		fmt.Println("Error resolving address:", err)
		os.Exit(1)
	}

	// 2. 监听 UDP 端口
	// ListenUDP 会绑定端口并准备接收数据
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("UDP Server started on %s\n", conn.LocalAddr().String())

	// 3. 循环接收数据
	buffer := make([]byte, 1024)
	for {
		// ReadFromUDP 会阻塞直到收到数据包
		// n: 读取的字节数
		// clientAddr: 发送者的地址 (IP + 端口)
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading:", err)
			continue
		}

		msg := string(buffer[:n])
		fmt.Printf("Received from %s: %s\n", clientAddr.String(), msg)

		// 4. 回复消息 (Echo)
		response := fmt.Sprintf("Server received: %s", msg)
		_, err = conn.WriteToUDP([]byte(response), clientAddr)
		if err != nil {
			fmt.Println("Error writing:", err)
		}
	}
}
