package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"photo-backup/model"
	"time"

	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type PhotoStorage interface {
	SavePhoto(ctx context.Context, fileHeader *multipart.FileHeader) error
}

type LocalPhotoStorage struct {
	Directory string
	Db        PhotoDB
	Log       *zap.Logger
}

func (s *LocalPhotoStorage) SavePhoto(ctx context.Context, fileHeader *multipart.FileHeader) error {
	if fileHeader == nil {
		s.Log.Error("file header is nil")
		return fmt.Errorf("file header cannot be nil")
	}

	file, err := fileHeader.Open()
	if err != nil {
		s.Log.Error("failed to open file", zap.Error(err))
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// create temp file
	tmpFile, err := os.CreateTemp("", "photo-*.tmp")
	if err != nil {
		s.Log.Error("failed to create temp file", zap.Error(err))
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpFilePath := tmpFile.Name()
	defer os.Remove(tmpFilePath)

	// copy uploaded file to temp file
	if _, err := io.Copy(tmpFile, file); err != nil {
		tmpFile.Close()
		s.Log.Error("failed to copy file to temp", zap.Error(err), zap.String("temp_path", tmpFilePath))
		return fmt.Errorf("failed to copy file to temp: %w", err)
	}

	// seek to beginning for EXIF reading
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		tmpFile.Close()
		s.Log.Error("failed to seek to start of temp file", zap.Error(err), zap.String("temp_path", tmpFilePath))
		return fmt.Errorf("failed to seek to start of temp file: %w", err)
	}

	// extract EXIF data
	var geoPoint model.GeoPoint
	var takenAt time.Time
	exifData, err := exif.Decode(tmpFile)
	if err != nil {
		s.Log.Warn("failed to decode EXIF data, using defaults", zap.Error(err))
		takenAt = time.Now()
	} else {
		if lat, long, err := exifData.LatLong(); err == nil {
			geoPoint = model.GeoPoint{
				Type:        "Point",
				Coordinates: []float64{long, lat},
			}
		}
		if tm, err := exifData.DateTime(); err == nil {
			takenAt = tm
		} else {
			takenAt = time.Now()
		}
	}

	// close temp file
	if err := tmpFile.Close(); err != nil {
		s.Log.Error("failed to close temp file", zap.Error(err), zap.String("temp_path", tmpFilePath))
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// determine file extension
	extension := filepath.Ext(fileHeader.Filename)
	if extension == "" {
		contentType := fileHeader.Header.Get("Content-Type")
		extensions, _ := mime.ExtensionsByType(contentType)
		if len(extensions) > 0 {
			extension = extensions[0]
			s.Log.Debug("using extension from content type", zap.String("extension", extension), zap.String("content_type", contentType))
		} else {
			extension = ".jpg"
			s.Log.Warn("no extension found, defaulting to .jpg", zap.String("content_type", contentType))
		}
	}

	// generate file names and paths
	id := primitive.NewObjectIDFromTimestamp(takenAt)
	fileName := id.Hex() + extension
	thumbName := id.Hex() + "_thumb" + extension
	filePath := filepath.Join(s.Directory, fileName)
	thumbPath := filepath.Join(s.Directory, thumbName)

	// move temp file to final location
	if err := os.Rename(tmpFilePath, filePath); err != nil {
		s.Log.Error("failed to move temp file", zap.Error(err), zap.String("file_path", filePath))
		return fmt.Errorf("failed to move temp file to %s: %w", filePath, err)
	}

	// generate thumbnail
	if err := generateThumbnail(filePath, thumbPath); err != nil {
		os.Remove(filePath) // Clean up main file
		s.Log.Error("failed to generate thumbnail", zap.Error(err), zap.String("thumb_path", thumbPath))
		return fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	// save to mongo
	photo := model.PhotoDB{
		ID:            id,
		Size:          fileHeader.Size,
		ContentType:   fileHeader.Header.Get("Content-Type"),
		FilePath:      filePath,
		ThumbnailPath: thumbPath,
		TakenAt:       takenAt,
		LonLat:        geoPoint,
	}
	if _, err := s.Db.SavePhoto(ctx, photo); err != nil {
		// clean up files if database save fails
		os.Remove(filePath)
		os.Remove(thumbPath)
		s.Log.Error("failed to save photo metadata to database", zap.Error(err), zap.String("file_path", filePath))
		return fmt.Errorf("failed to save photo metadata: %w", err)
	}

	s.Log.Info("photo saved successfully", zap.String("file_path", filePath), zap.String("photo_id", id.Hex()))
	return nil
}

func generateThumbnail(filePath, thumbnailPath string) error {
	src, err := imaging.Open(filePath, imaging.AutoOrientation(true))
	if err != nil {
		return err
	}

	dst := imaging.Fill(src, 100, 100, imaging.Center, imaging.Lanczos)
	err = imaging.Save(dst, thumbnailPath)
	if err != nil {
		return err
	}
	return nil
}
