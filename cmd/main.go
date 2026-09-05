package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	app, err := newAPI(ctx)
	if err != nil {
		panic(err)
	}
	defer app.pool.Close()

	const numWorkers = 3

	// Start background workers
	for i := 0; i < numWorkers; i++ {
		go app.worker.Run(ctx)
	}

	go app.scheduler.Run(ctx)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: app.router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	log.Printf("API up")
	<-ctx.Done()
	log.Printf("shutting down")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}
}
