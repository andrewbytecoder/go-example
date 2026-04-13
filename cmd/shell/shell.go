package main

import (
	"fmt"
	"os/exec"

	"mvdan.cc/sh/v3/shell"
)

func main() {
	exec.Command("bash", "-c", "ls -l")

	// 将对应的脚本命令解析成fields字段
	fields, err := shell.Fields("ls -l", nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(fields)
}
