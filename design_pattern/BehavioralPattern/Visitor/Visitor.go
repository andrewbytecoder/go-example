package main

import "fmt"

// 假设我们有一个文档对象模型（DOM），
// 包含多种元素（如文本、图片、表格等）。
// 我们需要对这些元素进行不同的操作（
// 如导出为 PDF、导出为 HTML 等）。

type Element interface {
	Accept(v Visitor)
}
type Text struct {
	content string
}

func (t *Text) Accept(v Visitor) {
	v.VisitText(t)
}

type Image struct {
	src string
}

func (t *Image) Accept(v Visitor) {
	v.VisitImage(t)
}

type Visitor interface {
	VisitText(text *Text)
	VisitImage(image *Image)
}
type PDFVisitor struct {
}

func (p PDFVisitor) VisitText(text *Text) {
	fmt.Println("pdf 文本内容", text.content)
}
func (p PDFVisitor) VisitImage(image *Image) {
	fmt.Println("pdf 获取图片的src", image.src)
}

type HTMLVisitor struct {
}

func (p HTMLVisitor) VisitText(text *Text) {
	fmt.Println("html 文本内容", text.content)
}
func (p HTMLVisitor) VisitImage(image *Image) {
	fmt.Println("html 获取图片的src", image.src)
}

type Document struct {
	elements []Element
}

func (d *Document) AddElement(element Element) {
	d.elements = append(d.elements, element)
}

func (d *Document) Accept(visitor Visitor) {
	for _, element := range d.elements {
		element.Accept(visitor)
	}
}

func main() {
	//将算法和对象数据结构分离
	// 内容是 document , 内容可以传递给PDF算法，也可以传递给html算法
	document := &Document{}

	document.AddElement(&Text{content: "你好"})
	document.AddElement(&Text{content: "xxx"})
	document.AddElement(&Image{src: "abc.png"})

	pdf := &PDFVisitor{}
	document.Accept(pdf)
	html := &HTMLVisitor{}
	document.Accept(html)
}
