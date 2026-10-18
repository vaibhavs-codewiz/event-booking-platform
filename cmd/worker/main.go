// Command worker is the background worker entry point for the Event Booking
// Platform.
//
// Phase 1: the worker is a placeholder. It starts, waits for a shutdown signal,
// and exits. Phase 2 will wire it to Asynq (backed by Redis) to process:
//   - booking confirmation email after a successful booking
//   - event update notification to all customers who booked that event
//
// Keeping a separate worker binary now means the deployment shape is stable:
// one API process handles HTTP, one worker process consumes the queue.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger := log.New(os.Stdout, "worker ", log.LstdFlags|log.LUTC)
	logger.Println("worker starting (phase 1: no tasks registered)")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	logger.Println("worker stopped")
}
