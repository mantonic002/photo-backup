package storage

import (
	"io"
	"log"
	"mime/multipart"
	"os"
	"photo-backup/model"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
)

type PhotoStorage interface {
	SavePhoto(fileHeader *multipart.FileHeader) error
}

type LocalPhotoStorage struct {
	Directory string
	Db        PhotoDB
}

func (s *LocalPhotoStorage) SavePhoto(fileHeader *multipart.FileHeader) error {
	filePath := s.Directory + "/" + fileHeader.Filename
	outFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	size, err := io.Copy(outFile, file)
	if err != nil {
		return err
	}

	exifFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer exifFile.Close()

	// extract exif data
	var geoPoint model.GeoPoint
	var takenAt time.Time
	// Reopen for EXIF
	x, err := exif.Decode(exifFile)
	if err != nil {
		log.Println("Error decoding EXIF data, proceeding without it:", err)
		return nil
	} else {
		// lat/long
		if lat, long, err := x.LatLong(); err == nil {
			geoPoint = model.GeoPoint{
				Type:        "Point",
				Coordinates: []float64{long, lat},
			}
		}
		// datetime
		if tm, err := x.DateTime(); err == nil {
			takenAt = tm
		}
	}

	thumbnailPath, err := generateThumbnail(filePath)
	if err != nil {
		return err
	}

	err = s.Db.SavePhoto(model.PhotoDB{
		Size:          size,
		ContentType:   fileHeader.Header.Get("Content-Type"),
		FilePath:      filePath,
		ThumbnailPath: thumbnailPath,
		TakenAt:       takenAt,
		LonLat:        geoPoint,
	})
	if err != nil {
		return err
	}

	return nil
}

func generateThumbnail(filePath string) (string, error) {
	src, err := imaging.Open(filePath, imaging.AutoOrientation(true))
	if err != nil {
		return "", err
	}

	splitPath := strings.Split(filePath, ".")
	fileExtension := splitPath[len(splitPath)-1]
	pathNoExtension := strings.Join(splitPath[0:len(splitPath)-1], ".")

	thumbnailPath := pathNoExtension + "_thumb." + fileExtension

	dst := imaging.Fill(src, 100, 100, imaging.Center, imaging.Lanczos)
	err = imaging.Save(dst, thumbnailPath)
	if err != nil {
		return "", err
	}
	return thumbnailPath, nil
}
