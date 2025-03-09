package main

import "fmt"

// IntSliceIterator 定义迭代器结构体
type IntSliceIterator struct {
	slice []int // 要迭代的slice
	index int   // 当前索引位置
}

type Iterator interface {
	HasNext() bool
	Next() (int, bool)
}

// NewIntSliceIterator 创建一个新的迭代器实例
func NewIntSliceIterator(slice []int) *IntSliceIterator {
	return &IntSliceIterator{slice: slice, index: -1}
}

// HasNext 判断是否还有下一个元素
func (it *IntSliceIterator) HasNext() bool {
	if it.index+1 < len(it.slice) {
		return true
	}
	return false
}

// Next 返回下一个元素，并将索引加一
func (it *IntSliceIterator) Next() (int, bool) {
	if !it.HasNext() {
		return 0, false
	}
	it.index++
	return it.slice[it.index], true
}

func main() {
	// 提供一种方法，顺序访问一个集合中的各个元素，而又不需要暴露该对象内部的实现方式
	// 核心思想就是将遍历的逻辑从集合的具体实现中分离出来
	// 比如将slice 和map向外提供同样的迭代器来对数据进行遍历
	numbers := []int{1, 2, 3, 4, 5}
	iterator := NewIntSliceIterator(numbers)

	for iterator.HasNext() {
		value, ok := iterator.Next()
		if !ok {
			break
		}
		fmt.Println(value)
	}
}
