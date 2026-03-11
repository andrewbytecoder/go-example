package main

import "github.com/go-example/json"

func main() {
	jsonMarshal()
}

func jsonMarshal() {
	err := json.Unmarshal()
	if err != nil {
		return
	}
}
