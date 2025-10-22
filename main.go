package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"photo-backup/api"
	"photo-backup/storage"
	"time"

	"github.com/gorilla/mux"
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

	secret := os.Getenv("JWT_SECRET")

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
	h := api.NewPhotoHandlers(localStorage, mongodb, secret, logger)
	r := mux.NewRouter()

	// PUBLIC ROUTES
	r.HandleFunc("/login", h.HandleLogin).Methods(http.MethodPost)

	// PROTECTED ROUTES
	protected := r.NewRoute().Subrouter()
	protected.HandleFunc("/photos", h.HandleGetPhoto).Queries("lastId", "{lastId}", "limit", "{limit}").Methods(http.MethodGet)
	protected.HandleFunc("/photos/search", h.HandleSearchPhoto).Queries("lastId", "{lastId}", "limit", "{limit}", "latMin", "{latMin}", "latMax", "{latMax}", "longMin", "{longMin}", "longMax", "{longMax}").Methods(http.MethodGet)
	protected.HandleFunc("/photos", h.HandleUploadPhoto).Methods(http.MethodPost)
	protected.HandleFunc("/photos", h.HandleDeletePhoto).Queries("id", "{id}").Methods(http.MethodDelete, http.MethodOptions)
	protected.HandleFunc("/photos/bulk-delete", h.HandleDeleteMultiplePhotos).Methods(http.MethodDelete, http.MethodOptions)
	protected.PathPrefix("/files/").Handler(http.StripPrefix("/files/", http.FileServer(http.Dir("./.uploads"))))

	// MIDDLEWARE
	// protected.Use(api.AuthMiddleware(secret, logger))

	r.Use(api.CORSMiddleware())
	r.Use(api.RecoveryMiddleware(logger))
	r.Use(api.RequestLoggerMiddleware(logger))

	// START SERVER
	logger.Info("Starting server on :8080")

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}
