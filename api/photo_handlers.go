package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"photo-backup/model"
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
	fmt.Fprintln(w, "Get Photo Handler")
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
		if err := h.processAndStoreFile(fileHeader); err != nil {
			http.Error(w, "Failed to save photo: "+err.Error(), http.StatusInternalServerError)
			return
		}
		log.Println("Uploaded:", fileHeader.Filename)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully"})
}

func (h *PhotoHandlers) processAndStoreFile(fileHeader *multipart.FileHeader) error {
	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	photo := model.Photo{
		Size:        fileHeader.Size,
		ContentType: fileHeader.Header.Get("Content-Type"),
		Filename:    fileHeader.Filename,
		FileContent: content,
	}

	if err := h.Storage.SavePhoto(photo); err != nil {
		return fmt.Errorf("save photo: %w", err)
	}

	return nil
}
