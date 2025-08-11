package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"photo-backup/model"
	"time"

	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PhotoStorage interface {
	SavePhoto(ctx context.Context, fileHeader *multipart.FileHeader) error
}

type LocalPhotoStorage struct {
	Directory string
	Db        PhotoDB
}

func (s *LocalPhotoStorage) SavePhoto(ctx context.Context, fileHeader *multipart.FileHeader) error {
	if fileHeader == nil {
		return fmt.Errorf("file header cannot be nil")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// temp file
	tmpFile, err := os.CreateTemp("", "photo-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpFilePath := tmpFile.Name()
	defer os.Remove(tmpFilePath)

	if _, err := io.Copy(tmpFile, file); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to copy file to temp: %w", err)
	}

	// close the temp file
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// open temp file for EXIF
	tmpFile, err = os.Open(tmpFilePath)
	if err != nil {
		return fmt.Errorf("failed to reopen temp file for EXIF: %w", err)
	}
	defer tmpFile.Close()

	// extract EXIF data
	var geoPoint model.GeoPoint
	var takenAt time.Time
	exifData, err := exif.Decode(tmpFile)
	if err != nil {
		log.Printf("Error decoding EXIF data, proceeding without it: %v", err)
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

	tmpFile.Close()

	// extract extension
	extension := filepath.Ext(fileHeader.Filename)
	if extension == "" {
		contentType := fileHeader.Header.Get("Content-Type")
		extensions, _ := mime.ExtensionsByType(contentType)
		if len(extensions) > 0 {
			extension = extensions[0]
		} else {
			extension = ".jpg"
		}
	}

	id := primitive.NewObjectIDFromTimestamp(takenAt)
	fileName := id.Hex() + extension
	thumbName := id.Hex() + "_thumb" + extension
	filePath := filepath.Join(s.Directory, fileName)
	thumbPath := filepath.Join(s.Directory, thumbName)

	// move temp file to final destination
	log.Printf("Attempting to rename %s to %s", tmpFilePath, filePath)
	if err := os.Rename(tmpFilePath, filePath); err != nil {
		return fmt.Errorf("failed to move temp file to %s: %w", filePath, err)
	}

	err = generateThumbnail(filePath, thumbPath)
	if err != nil {
		os.Remove(filePath) // clean up main file
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
		return fmt.Errorf("failed to save photo metadata: %w", err)
	}

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
