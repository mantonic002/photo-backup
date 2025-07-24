package main

import (
	"log"
	"net/http"
	"photo-backup/api"
)

func main() {
	apiHandlers := &api.PhotoHandlers{}
	mux := http.NewServeMux()
	apiHandlers.ServeHTTP(mux)

	log.Println("Starting server on :8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
