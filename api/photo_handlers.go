package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
)

type PhotoHandlers struct {
	// TODO: Storage
}

type Photo struct {
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	Filename    string `json:"filename"`
	FileContent string `json:"file_content"`
}

func (h *PhotoHandlers) ServeHTTP(mux *http.ServeMux) {
	mux.HandleFunc("/photos", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.handleGetPhoto(w, r)
		case http.MethodPost:
			h.handleUploadPhoto(w, r)
		default:
			log.Println("Unsupported method", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func (h *PhotoHandlers) handleGetPhoto(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Get Photo Handler")
}

func (h *PhotoHandlers) handleUploadPhoto(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		log.Fatalln(errors.New("invalid method"))
		return
	}

	var size int64 = 200 * 1024 * 1024 // 200 MB
	if r.ContentLength > size {
		log.Println("File size exceeds limit:", r.ContentLength, size)
		http.Error(w, "File size exceeds limit", http.StatusRequestEntityTooLarge)
		return
	}
	r.ParseMultipartForm(size)
	if r.MultipartForm == nil || len(r.MultipartForm.File) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	// process file.
	fileHeaders := r.MultipartForm.File["file"]
	if len(fileHeaders) == 0 {
		http.Error(w, "No file found in the request", http.StatusBadRequest)
		return
	}

	for _, fileHeader := range fileHeaders {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Error opening file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		// TODO: Read file and store

		log.Println("File uploaded:", fileHeader.Filename)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File uploaded successfully")
}
