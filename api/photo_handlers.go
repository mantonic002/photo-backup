package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"photo-backup/storage"
)

type PhotoHandlers struct {
	Storage storage.PhotoStorage
}

func (h *PhotoHandlers) ServeHTTP(mux *http.ServeMux) {
	mux.HandleFunc("/photos", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.handleGetPhoto(w, r)
		case http.MethodPost:
			h.handleUploadPhoto(w, r)
		default:
			log.Println("Unsupported method:", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func (h *PhotoHandlers) handleGetPhoto(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	photoDB, file, err := h.Storage.GetPhoto(id)
	if err != nil {
		http.Error(w, "Photo not found: "+err.Error(), http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", photoDB.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", photoDB.FilePath))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, file)
}

func (h *PhotoHandlers) handleUploadPhoto(w http.ResponseWriter, r *http.Request) {
	const maxSize = 200 * 1024 * 1024 // 200 MB

	if r.ContentLength > maxSize {
		log.Println("File size exceeds limit:", r.ContentLength)
		http.Error(w, "File size exceeds limit", http.StatusRequestEntityTooLarge)
		return
	}

	err := r.ParseMultipartForm(maxSize)
	if err != nil {
		http.Error(w, "Error parsing form: "+err.Error(), http.StatusBadRequest)
		return
	}

	fileHeaders := r.MultipartForm.File["file"]
	if len(fileHeaders) == 0 {
		http.Error(w, "No file found in the request", http.StatusBadRequest)
		return
	}

	for _, fileHeader := range fileHeaders {
		if err := h.Storage.SavePhoto(fileHeader); err != nil {
			http.Error(w, "Failed to save photo: "+err.Error(), http.StatusInternalServerError)
			return
		}
		log.Println("Uploaded:", fileHeader.Filename)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully"})
}
