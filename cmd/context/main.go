package main

import "github.com/go-example/context"

func main() {
	ctx, cancel := context.NewContext()

	ch := make(chan int, 100)
	// make 出来的channel 需要及时释放
	defer close(ch)
	context.Run(ctx, ch)

	for i := 0; i < 100; i++ {
		x := <-ch
		println(x)
	}
	cancel()
}
