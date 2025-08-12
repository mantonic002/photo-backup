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
	"go.uber.org/zap"
)

func main() {
	// EVOIRMENT
	_ = godotenv.Load(".env")
	_ = godotenv.Overload(".env.secret")

	mongoURI := os.Getenv("MONGO_URI")
	mongoDB := os.Getenv("MONGO_DB")
	mongoCollection := os.Getenv("MONGO_COLLECTION")

	// LOGGER
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	// CONTEXT
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// MONGO
	mongodb := &storage.MongoPhotoDB{}
	err = mongodb.Connect(ctx, logger, mongoURI, mongoDB, mongoCollection)
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB:",
			zap.String("action", "db_connection"),
			zap.Error(err),
		)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		mongodb.Close(closeCtx)
	}()

	// LOCAL STORAGE
	localStorage := &storage.LocalPhotoStorage{
		Directory: "./.uploads",
		Db:        mongodb,
		Log:       logger,
	}

	// HANDLERS
	apiHandlers := api.NewPhotoHandlers(localStorage, mongodb, logger)
	mux := http.NewServeMux()
	apiHandlers.ServeHTTP(mux)

	logger.Info("Starting server on :8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
