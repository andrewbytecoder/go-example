package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func shutdownHandler(ctx context.Context, sigs chan os.Signal, cancel context.CancelFunc) {
	// Wait for the context do be Done or for the signal to come in to shutdown.
	select {
	case <-ctx.Done():
		log.Println("Stopping shutdownHandler...")
	case <-sigs:
		// Call cancel on the context to close everything down.
		cancel()
		log.Println("shutdownHandler sent cancel signal...")
	}

	// Unregister to get default OS nuke behaviour in case we don't exit cleanly
	signal.Stop(sigs)
}
func main() {
	// This is the main context that everything should run in.
	// All spawned goroutines should exit when cancel is called on this context.
	// Go routines spawned from main.go coordinate using a WaitGroup. This provides a mechanism to allow the shutdownHandler goroutine
	// to block until all the goroutines return . If those goroutines spawn other goroutines then they are responsible for
	// blocking and returning only when cancel() is called.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}
	// Register for SIGINT and SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	wg.Add(1)
	go func() {
		shutdownHandler(ctx, sigs, cancel)
		wg.Done()
	}()

	// wait for all goroutines to return
	wg.Wait()

}
