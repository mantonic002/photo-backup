package api

import (
	"encoding/json"
	"net/http"
	"os"
	"photo-backup/storage"
	"strconv"

	"go.uber.org/zap"
)

type PhotoHandlers struct {
	Storage   storage.PhotoStorage
	Db        storage.PhotoDB
	SecretKey string
	Log       *zap.Logger
}

type LoginRequest struct {
	Password string `json:"password"`
}

func NewPhotoHandlers(storage storage.PhotoStorage, db storage.PhotoDB, logger *zap.Logger) *PhotoHandlers {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		logger.Fatal("JWT_SECRET is empty")
	}
	return &PhotoHandlers{
		Storage:   storage,
		Db:        db,
		SecretKey: secret,
		Log:       logger,
	}
}

func (h *PhotoHandlers) ServeHTTP(mux *http.ServeMux) {
	mux.HandleFunc("/login", RequestLoggerMiddleware(h.Log, recoveryMiddleware(h.handleLogin)))

	mux.HandleFunc("/photos",
		RequestLoggerMiddleware(h.Log, recoveryMiddleware(
			h.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					h.handleGetPhoto(w, r)
				case http.MethodPost:
					h.handleUploadPhoto(w, r)
				default:
					h.Log.Error("unsupported HTTP method", zap.String("method", r.Method), zap.String("path", r.URL.Path))
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
			}))))

	mux.HandleFunc("/photos/search",
		RequestLoggerMiddleware(h.Log, recoveryMiddleware(
			h.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					h.handleSearchPhoto(w, r)
				} else {
					h.Log.Error("unsupported HTTP method", zap.String("method", r.Method), zap.String("path", r.URL.Path))
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
			}))))

	mux.Handle("/files/",
		RequestLoggerMiddleware(h.Log, recoveryMiddleware(
			h.authMiddleware(
				http.StripPrefix("/files/", http.FileServer(http.Dir("./.Uploads"))).ServeHTTP,
			),
		)))
}

func (h *PhotoHandlers) handleGetPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.URL.Query().Get("id")
	lastId := r.URL.Query().Get("lastId")
	limitStr := r.URL.Query().Get("limit")

	if id == "" && limitStr == "" {
		h.Log.Error("missing necessary parameters", zap.String("path", r.URL.Path))
		http.Error(w, "Missing necessary parameters", http.StatusBadRequest)
		return
	}

	if id != "" { // single photo, return full size photo
		photo, err := h.Db.GetPhoto(ctx, id)
		if err != nil {
			h.Log.Info("photo not found", zap.String("photo_id", id), zap.Error(err))
			http.Error(w, "Photo not found: "+err.Error(), http.StatusNotFound)
			return
		}

		h.Log.Info("retrieved photo", zap.String("photo_id", id), zap.String("file_path", photo.FilePath))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(photo)

	} else if limitStr != "" { // multiple photos, return thumbnails
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			h.Log.Error("invalid limit value", zap.String("limit", limitStr), zap.Error(err))
			http.Error(w, "Invalid limit value", http.StatusBadRequest)
			return
		}

		photos, err := h.Db.GetPhotos(ctx, lastId, int64(limit))
		if err != nil {
			h.Log.Info("failed to fetch photos", zap.String("last_id", lastId), zap.Int64("limit", int64(limit)), zap.Error(err))
			http.Error(w, "Failed to fetch photos: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if len(photos) == 0 {
			h.Log.Warn("no photos found", zap.String("last_id", lastId), zap.Int64("limit", int64(limit)))
			http.Error(w, "No photos found", http.StatusNotFound)
			return
		}

		h.Log.Info("retrieved photos", zap.Int("count", len(photos)), zap.String("last_id", lastId), zap.Int64("limit", int64(limit)))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(photos)
	}
}

func (h *PhotoHandlers) handleUploadPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	const maxSize = 200 * 1024 * 1024 // 200 MB

	if r.ContentLength > maxSize {
		h.Log.Error("file size exceeds limit", zap.Int64("content_length", r.ContentLength), zap.Int64("max_size", maxSize))
		http.Error(w, "File size exceeds limit", http.StatusRequestEntityTooLarge)
		return
	}

	err := r.ParseMultipartForm(maxSize)
	if err != nil {
		h.Log.Error("failed to parse multipart form", zap.Error(err))
		http.Error(w, "Error parsing form: "+err.Error(), http.StatusBadRequest)
		return
	}

	fileHeaders := r.MultipartForm.File["file"]
	if len(fileHeaders) == 0 {
		h.Log.Error("no file found in request", zap.String("path", r.URL.Path))
		http.Error(w, "No file found in the request", http.StatusBadRequest)
		return
	}

	for _, fileHeader := range fileHeaders {
		if err := h.Storage.SavePhoto(ctx, fileHeader); err != nil {
			h.Log.Error("failed to save photo", zap.String("filename", fileHeader.Filename), zap.Error(err))
			http.Error(w, "Failed to save photo: "+err.Error(), http.StatusInternalServerError)
			return
		}
		h.Log.Info("photo uploaded successfully", zap.String("filename", fileHeader.Filename))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully"})
}

func (h *PhotoHandlers) handleSearchPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	longStr := r.URL.Query().Get("long")
	latStr := r.URL.Query().Get("lat")
	distStr := r.URL.Query().Get("dist")
	if longStr == "" || latStr == "" || distStr == "" {
		h.Log.Error("missing search parameters", zap.String("long", longStr), zap.String("lat", latStr), zap.String("dist", distStr))
		http.Error(w, "Missing parameter", http.StatusBadRequest)
		return
	}

	long, err := strconv.ParseFloat(longStr, 64)
	if err != nil {
		h.Log.Info("invalid longitude value", zap.String("long", longStr), zap.Error(err))
		http.Error(w, "Invalid longitude value", http.StatusBadRequest)
		return
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		h.Log.Info("invalid latitude value", zap.String("lat", latStr), zap.Error(err))
		http.Error(w, "Invalid latitude value", http.StatusBadRequest)
		return
	}
	dist, err := strconv.Atoi(distStr)
	if err != nil {
		h.Log.Info("invalid distance value", zap.String("dist", distStr), zap.Error(err))
		http.Error(w, "Invalid distance value", http.StatusBadRequest)
		return
	}

	photos, err := h.Db.SearchPhotosByLocation(ctx, long, lat, dist)
	if err != nil {
		h.Log.Error("failed to search photos by location", zap.Error(err))
		http.Error(w, "Failed to fetch photos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(photos) == 0 {
		h.Log.Info("no photos found for location search")
		http.Error(w, "No photos found", http.StatusNotFound)
		return
	}

	h.Log.Info("retrieved photos by location", zap.Int("count", len(photos)))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(photos)
}
