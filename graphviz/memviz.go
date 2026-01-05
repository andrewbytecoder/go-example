package main

import (
	"bytes"
	"os"

	"github.com/bradleyjkemp/memviz"
)

type node struct {
	id int
}

type tree struct {
	id    int
	left  *tree
	right *tree
	node  *node
}

func main() {
	root := &tree{
		id: 0,
		left: &tree{
			id: 1,
		},
		right: &tree{
			id: 2,
		},
	}
	leaf := &tree{
		id: 3,
	}

	root.node = &node{id: 0}

	root.left.right = leaf
	root.right.left = leaf

	buf := &bytes.Buffer{}
	memviz.Map(buf, &root)
	err := os.WriteFile("example-tree-data", buf.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
}
