package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"photo-backup/api"
	"photo-backup/storage"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	mongoURI := os.Getenv("MONGO_URI")
	mongoDB := os.Getenv("MONGO_DB")
	mongoCollection := os.Getenv("MONGO_COLLECTION")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongodb := &storage.MongoPhotoDB{}
	err := mongodb.Connect(ctx, mongoURI, mongoDB, mongoCollection)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		mongodb.Close(closeCtx)
	}()

	localStorage := &storage.LocalPhotoStorage{
		Directory: "./.uploads",
		Db:        mongodb,
	}

	apiHandlers := api.NewPhotoHandlers(localStorage, mongodb)
	mux := http.NewServeMux()
	apiHandlers.ServeHTTP(mux)

	log.Println("Starting server on :8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
