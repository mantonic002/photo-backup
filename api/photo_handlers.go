package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"photo-backup/storage"
	"strconv"
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

	mux.HandleFunc("/photos/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.handleSearchPhoto(w, r)
		} else {
			log.Println("Unsupported method:", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func (h *PhotoHandlers) handleGetPhoto(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	lastId := r.URL.Query().Get("lastId")
	limitStr := r.URL.Query().Get("limit")

	if id == "" && limitStr == "" {
		http.Error(w, "Missing necessary parameters", http.StatusBadRequest)
		return
	}

	if id != "" {
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
	} else if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "Invalid distance value", http.StatusBadRequest)
			return
		}
		h.Storage.GetPhotos(lastId, int64(limit))
	}
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

func (h *PhotoHandlers) handleSearchPhoto(w http.ResponseWriter, r *http.Request) {
	longStr := r.URL.Query().Get("long")
	latStr := r.URL.Query().Get("lat")
	distStr := r.URL.Query().Get("dist")
	if longStr == "" || latStr == "" || distStr == "" {
		http.Error(w, "Missing parameter", http.StatusBadRequest)
		return
	}

	long, err := strconv.ParseFloat(longStr, 64)
	if err != nil {
		http.Error(w, "Invalid longitude value", http.StatusBadRequest)
		return
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		http.Error(w, "Invalid latitude value", http.StatusBadRequest)
		return
	}
	dist, err := strconv.Atoi(distStr)
	if err != nil {
		http.Error(w, "Invalid distance value", http.StatusBadRequest)
		return
	}

	h.Storage.SearchPhotosByLocation(long, lat, dist)
}
