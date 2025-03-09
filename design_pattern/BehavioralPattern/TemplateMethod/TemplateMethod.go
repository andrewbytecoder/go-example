package main

import "fmt"

type Template interface {
	BoilWater()        // 烧水
	Brew()             // 冲泡
	AddSugar()         // 加糖
	HasAddSugar() bool // 是否加糖
}

type Coffee struct {
}

func (Coffee) BoilWater() { fmt.Println("烧水") }
func (Coffee) Brew()      { fmt.Println("冲泡") }
func (Coffee) AddSugar()  { fmt.Println("加糖") }
func (Coffee) HasAddSugar() bool {
	return true
}

type Tea struct {
}

func (Tea) BoilWater() { fmt.Println("烧水") }
func (Tea) Brew()      { fmt.Println("冲泡") }
func (Tea) AddSugar()  { fmt.Println("加糖") }
func (Tea) HasAddSugar() bool {
	return false
}

func MakeTemplate(tmp Template) {
	tmp.BoilWater()
	tmp.Brew()
	if tmp.HasAddSugar() {
		tmp.AddSugar()
	}
}

func main() {
	// 核心思想就是父类只提供算法的骨架，具体实现细节放到子类中进行思想
	// 就是将公用部分抽象实现出来
	// STL 中也大量使用了模板模式
	tea := Tea{}
	coffee := Coffee{}
	MakeTemplate(tea)
	MakeTemplate(coffee)
}
