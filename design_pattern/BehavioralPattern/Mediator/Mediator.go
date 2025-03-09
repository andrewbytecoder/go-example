package main

import "fmt"

type Obj interface {
	SendMsg(string)
	RevMsg(string)
}
type Mediator interface {
	SendMsg(msg string, user Obj)
}

type User struct {
	Name     string
	mediator Mediator
}

func (u User) SendMsg(msg string) {
	fmt.Printf("用户 %s 发了消息 %s\n", u.Name, msg)
	u.mediator.SendMsg(msg, u)
}
func (u User) RevMsg(msg string) {
	fmt.Printf("用户 %s 接收到消息 %s\n", u.Name, msg)
}

type ChatRoom struct {
	users []User
}

func (c *ChatRoom) Register(user User) {
	c.users = append(c.users, user)
}
func (c *ChatRoom) SendMsg(msg string, user Obj) {
	for _, u := range c.users {
		if u == user {
			continue
		}
		u.RevMsg(msg)
	}
}

func main() {
	// 将双方之间的交付复杂过程都集中到中介着身上，这样能有效减少双方之间的耦合
	// 开发IM等通信模式下通常会使用到该模式
	// 但发送消息，或者存在集群概念的情况下通常需要使用该模式
	// 如果不使用A B C 之间想要发送消息就需要都知道对方，然而通过中介者就可以将消息丢给中介 聊天室就可以了
	// 其实消息中间件就是一个中介者模式的具体实现
	room := ChatRoom{}
	u1 := User{Name: "枫枫", mediator: &room}
	u2 := User{Name: "张三", mediator: &room}
	u3 := User{Name: "李四", mediator: &room}

	room.Register(u1)
	room.Register(u2)
	room.Register(u3)

	u1.SendMsg("你好啊")
	u2.SendMsg("吃了吗")
	u3.SendMsg("我吃了")
}
