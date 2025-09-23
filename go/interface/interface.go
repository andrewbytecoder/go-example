package main

import (
	"log"
)

type NexaCommand struct {
}

func (n *NexaCommand) AddCommand() {
	log.Println("hello world")
}

func main() {
	var nexaCommand interface{}
	nexaCommand = &NexaCommand{}

	// 在go中任意实现了 AddCommand() 方法的对象都能转化为 CmdRegister 类型的数据
	// 如果将CmdRegister接口放到全局会自动显示方法实现，就算不放到全局在运行期间也能定义类型进行方法的连接
	type CmdRegister interface {
		AddCommand()
	}

	if cmdRegister, ok := nexaCommand.(CmdRegister); ok {
		cmdRegister.AddCommand()
	}

}
