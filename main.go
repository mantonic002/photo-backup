package main

import (
	"log"
	"net/http"
	"photo-backup/api"
	"photo-backup/storage"
)

func main() {
	mongodb := &storage.MongoPhotoDB{}
	err := mongodb.Connect("mongodb://localhost:27017", "photo_backup", "photos")
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer mongodb.Close()

	localStorage := &storage.LocalPhotoStorage{
		Directory: "./.uploads",
		Db:        mongodb,
	}

	apiHandlers := &api.PhotoHandlers{
		Storage: localStorage,
		Db:      mongodb,
	}
	mux := http.NewServeMux()
	apiHandlers.ServeHTTP(mux)

	log.Println("Starting server on :8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
