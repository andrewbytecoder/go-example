package main

import "fmt"

type State interface {
	Switch(context *Context)
}

type Context struct {
	state State
}

func (c *Context) SetState(state State) {
	c.state = state
}
func (c *Context) Switch() {
	c.state.Switch(c)
}

type OnState struct {
}

func (OnState) Switch(context *Context) {
	fmt.Println("开关关闭")
	context.SetState(&OffState{})
}

type OffState struct {
}

func (OffState) Switch(context *Context) {
	fmt.Println("开关打开")
	context.SetState(&OnState{})
}

func main() {
	// 允许对象内部状态改变时改变其行为，比如这里的switch调用状态不同时执行的函数行为也不同
	// 多使用状态模式能极大的避免在代码中使用if判断，增加代码的可读性
	c := &Context{
		state: OffState{},
	}
	c.Switch()
	c.Switch()
	c.Switch()
}
