package main

import "fmt"

// Printer 打印机
type Printer interface {
	PrintFile(file string) // 打印文件
}

type Epson struct {
}

func (Epson) PrintFile(file string) {
	fmt.Println("使用爱普生打印机打印文件")
}

type Hp struct {
}

func (Hp) PrintFile(file string) {
	fmt.Println("使用惠普打印机打印文件")
}

// Computer 电脑
type Computer interface {
	Print(string)       // 打印
	SetPrinter(Printer) // 设置打印机
}

type Mac struct {
	printer Printer
}

func (m *Mac) Print(file string) {
	// 电脑调打印机的打印方法
	fmt.Println("使用mac电脑")
	m.printer.PrintFile(file)
}

func (m *Mac) SetPrinter(printer Printer) {
	m.printer = printer
}

type Windows struct {
	printer Printer
}

func (m *Windows) Print(file string) {
	// 电脑调打印机的打印方法
	fmt.Println("使用windows电脑")
	m.printer.PrintFile(file)
}

func (m *Windows) SetPrinter(printer Printer) {
	m.printer = printer
}

func main() {
	// 通过interface实现对不同对象的抽象
	// 调用对象抽象，被调用对象也抽象，就实现了桥接模式
	// 比如理由两个电脑，两个打印机，那么电脑和打印机之间可以相互关联
	// 但是抽象之后，只需要将两个interface关联就行了，在各种开源代码被广泛的使用到了
	w := Windows{}
	//hp := Hp{}
	ep := Epson{}

	w.SetPrinter(ep)
	w.Print("xx")
}
