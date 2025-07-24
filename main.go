package main

import (
	"log"
	"net/http"
	"photo-backup/api"
	"photo-backup/storage"
)

func main() {
	localStorage := &storage.LocalPhotoStorage{
		Directory: "./.uploads",
	}

	apiHandlers := &api.PhotoHandlers{
		Storage: localStorage,
	}
	mux := http.NewServeMux()
	apiHandlers.ServeHTTP(mux)

	log.Println("Starting server on :8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
