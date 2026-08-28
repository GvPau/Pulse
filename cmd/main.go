package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"pulse/internal/database"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		panic("DATABASE_URL environment variable is not set")
	}

	pool, err := database.Connect(ctx, dsn)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	log.Printf("Database up")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	log.Printf("API up")
	http.ListenAndServe(":8080", r)

}
