package main

import "fmt"

type Node interface {
	Display(ident string)
}

type File struct {
	Name string
}

func (n File) Display(ident string) {
	fmt.Println(ident + n.Name)
}

type Dir struct {
	Name     string
	children []Node
}

func (n Dir) Display(ident string) {
	fmt.Println(ident + n.Name)
	for _, child := range n.children {
		child.Display(ident + "  ")
	}
}

func main() {
	// 文件和文件节点都是文件，都实现了Display
	// 只是文件夹的display中还会遍历子文件，但是文件本身没有文件夹了所有就没有遍历过程了
	// 对于多种功能类似，但是需要进行组合的实现中非常有用
	// 特别是linux中文件系统或者遍历k8s文件对象非常有用
	root := Dir{
		Name: "CreationalPatterns",
		children: []Node{
			Dir{
				Name: "AbstractFactory",
				children: []Node{
					File{
						Name: "AbstractFactory.go",
					},
				},
			},
			File{
				Name: "main.go",
			},
		},
	}
	root.Display("")
}
