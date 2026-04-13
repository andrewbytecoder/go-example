package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	// 直接使用注册的信号管控context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	_, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8080", http.NoBody)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
}
