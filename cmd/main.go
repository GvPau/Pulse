package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	// Start background workers
	go app.worker.Run(ctx)
	go app.scheduler.Run(ctx)

	log.Printf("API up")
	http.ListenAndServe(":8080", app.router)
}
