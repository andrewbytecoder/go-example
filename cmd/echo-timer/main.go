// Command echo-timer demonstrates the uvgo subset: TCP, UDP, timer, fs-poll, loop.Now().
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

	addr := "127.0.0.1:7379"
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

	tmp, err := os.CreateTemp("", "uvgo-fspoll-*")
	if err != nil {
		log.Fatal(err)
	}
	path := tmp.Name()
	if _, err := tmp.Write([]byte("v0")); err != nil {
		tmp.Close()
		log.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		log.Fatal(err)
	}
	defer os.Remove(path)

	if _, err := loop.FSPollStart(path, 250, func(_ *uvgo.FSPoll, status int, prev, curr *uvgo.StatT) {
		if status != 0 {
			log.Printf("fs-poll status=%d (uv_fs_poll_t error path)", status)
			return
		}
		log.Printf("fs-poll file changed size %d -> %d (prev mtime=%v curr=%v)",
			prev.Size, curr.Size, prev.ModTime, curr.ModTime)
	}); err != nil {
		log.Fatal(err)
	}
	time.AfterFunc(2*time.Second, func() {
		if err := os.WriteFile(path, []byte("v1-longer"), 0644); err != nil {
			log.Printf("fs-poll demo write: %v", err)
		}
	})

	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:7377")
	if err != nil {
		log.Fatal(err)
	}
	udp, err := loop.ListenUDP("udp", udpAddr, func(_ *uvgo.UDP, nread int, buf []byte, from *net.UDPAddr, _ uvgo.UDPFlags) {
		if nread < 0 {
			log.Printf("udp recv error errno~%d", -nread)
			return
		}
		log.Printf("udp recv %q from %v (loop.Now=%dms)", string(buf), from, loop.Now())
	})
	if err != nil {
		log.Fatal(err)
	}
	localUDP := udp.LocalAddr().(*net.UDPAddr)
	time.AfterFunc(400*time.Millisecond, func() {
		udp.Send(localUDP, []byte("ping"), func(e error) {
			if e != nil {
				log.Printf("udp send: %v", e)
			}
		})
	})

	log.Printf("listening tcp://%s udp://%s; fs-poll %s (Ctrl+C to stop)", addr, localUDP.String(), path)

	if err := loop.Run(); err != nil {
		log.Fatal(err)
	}
	log.Println("shutdown complete")
}
