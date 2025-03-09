package main

import (
	"fmt"
	"strings"
)

func main() {
	// 定义模板字符串
	// 定义一种语言的表示方式，并提供一个解释器件解释这个语言
	// 将自己的语言定义为一个抽象的语法树，然后使用抽象的语法树解释自己的语言
	const tmpl = `Hello, {{ Name }}! You are {{Age}} years old.`
	template := ParseTemplate(tmpl)
	res := template.Interpreter(&Context{
		Data: map[string]any{
			"Name": "fengfeng",
			"Age":  21,
		},
	})
	fmt.Println(res)
}

type Context struct {
	Data map[string]any
}

type Node interface {
	Interpreter(*Context) string
}

type TextNode struct {
	Content string
}

func (t *TextNode) Interpreter(ctx *Context) string {
	return t.Content
}

type VarNode struct {
	Key string
}

func (t *VarNode) Interpreter(ctx *Context) string {
	val, ok := ctx.Data[t.Key]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

type Template struct {
	tree []Node
}

func ParseTemplate(tmpl string) *Template {
	var template = new(Template)
	var index = 0
	for {
		startIndex := strings.Index(tmpl[index:], "{{")
		if startIndex == -1 {
			template.tree = append(template.tree, &TextNode{
				Content: tmpl[index:],
			})
			break
		}
		template.tree = append(template.tree, &TextNode{
			Content: tmpl[index : index+startIndex],
		})
		endIndex := strings.Index(tmpl[index+startIndex:], "}}")
		if endIndex == -1 {
			break
		}
		key := strings.TrimSpace(tmpl[index+startIndex+2 : index+startIndex+endIndex])
		template.tree = append(template.tree, &VarNode{
			Key: key,
		})
		index = index + startIndex + endIndex + 2
	}
	return template
}
func (t *Template) Interpreter(ctx *Context) string {
	var s string
	for _, node := range t.tree {
		s += node.Interpreter(ctx)
	}
	return s
}
