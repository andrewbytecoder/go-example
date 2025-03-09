package main

import "fmt"

type House struct {
	Door   string
	Window string
}

type HouseBuilder interface {
	BuildDoor(val string)
	BuildWindow(val string)
	GetHouse() *House
}

type Bao struct {
	house *House
}

func NewBao() *Bao {
	return &Bao{
		house: &House{},
	}
}

func (b *Bao) BuildDoor(val string) {
	b.house.Door = val
	fmt.Println("门建造成功")
}

func (b *Bao) BuildWindow(val string) {
	b.house.Window = val
	fmt.Println("窗户建造成功")
}
func (b *Bao) GetHouse() *House {
	return b.house
}

type Boos struct {
	builder HouseBuilder
}

func NewBoos(bao *Bao) *Boos {
	return &Boos{
		builder: bao,
	}
}

func (b Boos) GetHouse() *House {
	b.builder.BuildDoor("门")
	b.builder.BuildWindow("窗")
	return b.builder.GetHouse()
}

func main() {
	// 真正干活的是包工头
	// 创建一个经理 boos 让经历管理包工头进行干活
	// 原理就是一个监工，一个真正会干活的包工头
	b := NewBao()
	boos := NewBoos(b)
	fmt.Println(boos.GetHouse())
}
