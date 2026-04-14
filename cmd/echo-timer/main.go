// Command echo-timer demonstrates the uvgo subset: TCP echo server + periodic timer.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-example/uvgo"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	loop := uvgo.NewLoop(ctx)

	addr := "127.0.0.1:7373"
	if err := loop.ListenTCP("tcp", addr, func(c net.Conn) {
		defer c.Close()
		if _, err := io.Copy(c, c); err != nil && err != io.EOF {
			log.Printf("echo: %v", err)
		}
	}); err != nil {
		log.Fatal(err)
	}

	loop.TimerStart(0, time.Second, func() {
		fmt.Println("tick (uv_timer_t-style repeat)")
	})

	log.Printf("listening on tcp://%s (Ctrl+C to stop)", addr)

	if err := loop.Run(); err != nil {
		log.Fatal(err)
	}
	log.Println("shutdown complete")
}
