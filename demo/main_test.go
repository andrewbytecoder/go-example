// 文件路径: main_test.go
package main

import "testing"

type Data struct {
	data [204800]byte
}

func BenchmarkRangeWithCopy(b *testing.B) {
	for b.Loop() {
		var data []Data
		for i := 0; i < 100; i++ {
			data = append(data, Data{})
		}
		for _, d := range data {
			_ = d.data[1]
		}
	}

}

func BenchmarkRangeWithIndex(b *testing.B) {
	for b.Loop() {
		var data []Data
		for i := 0; i < 100; i++ {
			data = append(data, Data{})
		}
		for i := range data {
			_ = data[i].data[1]
		}
	}

}
