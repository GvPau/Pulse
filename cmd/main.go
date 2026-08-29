package main

import (
	"context"
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	ctx := context.Background()

	app, err := newAPI(ctx)
	if err != nil {
		panic(err)
	}

	defer app.pool.Close()

	log.Printf("API up")
	http.ListenAndServe(":8080", app.router)
}
