package main

import "fmt"

type Node interface {
	SetState(state string)
	GetState() string
}

type TextNode struct {
	state string
}

func (t *TextNode) SetState(state string) {
	t.state = state
}
func (t *TextNode) GetState() string {
	return t.state
}
func (t *TextNode) Save() Memento {
	return &TextMemento{
		state: t.state,
	}
}

type Memento interface {
	GetState() string
}
type TextMemento struct {
	// 这里是将Node的内容copy了一下
	state string
}

func (t TextMemento) GetState() string {
	return t.state
}

type Manage struct {
	states []Memento
}

func (m *Manage) Save(t Memento) {
	m.states = append(m.states, t)
}
func (m *Manage) Back(index int) Memento {
	return m.states[index]
}

func main() {
	// 就是在不破坏封装性的情况下，将状态恢复到指定的状态
	manage := Manage{}
	text := TextNode{}
	text.SetState("1")
	manage.Save(text.Save())
	text.SetState("2")
	manage.Save(text.Save())
	fmt.Println(text.GetState())
	fmt.Println(manage.Back(0).GetState())
}
