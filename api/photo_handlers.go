package api

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"sync"

	"net/http"
	"photo-backup/storage"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type PhotoHandlers struct {
	Storage   storage.PhotoStorage
	Db        storage.PhotoDB
	SecretKey string
	Log       *zap.Logger
}

func NewPhotoHandlers(storage storage.PhotoStorage, db storage.PhotoDB, secret string, logger *zap.Logger) *PhotoHandlers {
	return &PhotoHandlers{
		Storage:   storage,
		Db:        db,
		SecretKey: secret,
		Log:       logger,
	}
}

// GET
func (h *PhotoHandlers) HandleGetPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	lastId := vars["lastId"]
	limitStr := vars["limit"]

	if limitStr == "" {
		h.Log.Error("missing necessary parameters", zap.String("path", r.URL.Path))
		http.Error(w, "Missing necessary parameters", http.StatusBadRequest)
		return
	}

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

// UPLOAD
func (h *PhotoHandlers) HandleUploadPhoto(w http.ResponseWriter, r *http.Request) {
    const maxSize = 200 * 1024 * 1024 // 200 MB
    const maxConcurrentUploads = 10

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

    type uploadResult struct {
        Filename string
        Error    error
    }

    var wg sync.WaitGroup
    results := make(chan uploadResult, len(fileHeaders))
    sem := make(chan struct{}, maxConcurrentUploads)
    ctx := r.Context()

    for _, fileHeader := range fileHeaders {
        sem <- struct{}{} // Acquire semaphore
        wg.Add(1)
        go func(fileHeader *multipart.FileHeader) {
            defer wg.Done()
            defer func() { <-sem }() // Release semaphore

            select {
            case <-ctx.Done():
                h.Log.Warn("upload canceled", zap.String("filename", fileHeader.Filename), zap.Error(ctx.Err()))
                results <- uploadResult{Filename: fileHeader.Filename, Error: ctx.Err()}
                return
            default:
                if err := h.Storage.SavePhoto(ctx, fileHeader); err != nil {
                    h.Log.Error("failed to save photo", zap.String("filename", fileHeader.Filename), zap.Error(err))
                    results <- uploadResult{Filename: fileHeader.Filename, Error: err}
                    return
                }
                h.Log.Info("photo uploaded successfully", zap.String("filename", fileHeader.Filename))
                results <- uploadResult{Filename: fileHeader.Filename}
            }
        }(fileHeader)
    }

    wg.Wait()
    close(results)

    var successList, failedList []string
    for result := range results {
        if result.Error != nil {
            failedList = append(failedList, result.Filename)
            continue
        }
        successList = append(successList, result.Filename)
    }

    response := map[string]interface{}{
        "message":    "Photo upload completed",
        "successful": successList,
        "failed":     failedList,
        "count":      len(successList),
    }

    w.Header().Set("Content-Type", "application/json")
    if len(failedList) > 0 && len(successList) > 0 {
        w.WriteHeader(http.StatusMultiStatus)
    } else if len(failedList) > 0 {
        w.WriteHeader(http.StatusInternalServerError)
    } else {
        w.WriteHeader(http.StatusOK)
    }
    json.NewEncoder(w).Encode(response)
}

// DELETE SINGLE
func (h *PhotoHandlers) HandleDeletePhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	id := vars["id"]
	
	err := h.deletePhoto(ctx, id)
	if err != nil {
		h.Log.Error("failed to delete photo", zap.String("photo_id", id), zap.Error(err))
		return
	}

	h.Log.Info("photo deleted successfully", zap.String("photo_id", id))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Photo deleted successfully"})
}

// DELETE MULTIPLE
func (h *PhotoHandlers) HandleDeleteMultiplePhotos(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    var req struct {
        IDs []string `json:"ids"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.Log.Error("failed to decode delete multiple request", zap.Error(err))
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    if len(req.IDs) == 0 {
        h.Log.Error("no photo IDs provided for bulk delete")
        http.Error(w, "No photo IDs provided", http.StatusBadRequest)
        return
    }

    var failed []string
    for _, id := range req.IDs {
        if err := h.deletePhoto(ctx, id); err != nil {
            failed = append(failed, id)
            h.Log.Error("failed to delete photo in bulk", zap.String("photo_id", id), zap.Error(err))
        }
    }

    if len(failed) > 0 {
        http.Error(w, "Failed to delete some photos: "+fmt.Sprint(failed), http.StatusInternalServerError)
        return
    }

    h.Log.Info("deleted multiple photos", zap.Int("count", len(req.IDs)))
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"message": "Photos deleted successfully"})
}

func (h *PhotoHandlers) deletePhoto(ctx context.Context, id string) error {
	if id == "" {
		h.Log.Error("missing photo ID parameter")
		return fmt.Errorf("missing photo ID parameter")
	}

	err := h.Storage.DeletePhoto(ctx, id)
	if err != nil {
		h.Log.Error("failed to delete photo", zap.String("photo_id", id), zap.Error(err))
		return err
	}
	return nil
}

// SEARCH
func (h *PhotoHandlers) HandleSearchPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	lastId := vars["lastId"]
	limitStr := vars["limit"]
	longStr := vars["long"]
	latStr := vars["lat"]
	distStr := vars["dist"]

	if limitStr == "" {
		h.Log.Error("missing necessary parameters", zap.String("path", r.URL.Path))
		http.Error(w, "Missing necessary parameters", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		h.Log.Error("invalid limit value", zap.String("limit", limitStr), zap.Error(err))
		http.Error(w, "Invalid limit value", http.StatusBadRequest)
		return
	}

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

	photos, err := h.Db.SearchPhotosByLocation(ctx, lastId, int64(limit), long, lat, dist)
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
