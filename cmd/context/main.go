package main

import (
	"context"
	"time"
)

func main() {
	ctx := context.Background()
	dCtx, cancel := context.WithCancelCause(ctx)

}
