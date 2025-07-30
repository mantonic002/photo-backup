package storage

import (
	"io"
	"log"
	"mime/multipart"
	"os"
	"photo-backup/model"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

type PhotoStorage interface {
	SavePhoto(fileHeader *multipart.FileHeader) error
	GetPhoto(id string) (*model.PhotoDB, *os.File, error)
	GetPhotos(lastIdString string, limit int64) error
	SearchPhotosByLocation(long float64, lat float64, dist int) error
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

	err = s.Db.SavePhoto(model.PhotoDB{
		Size:        size,
		ContentType: fileHeader.Header.Get("Content-Type"),
		FilePath:    filePath,
		TakenAt:     takenAt,
		LonLat:      geoPoint,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *LocalPhotoStorage) GetPhoto(id string) (*model.PhotoDB, *os.File, error) {
	photoDB, err := s.Db.GetPhoto(id)
	if err != nil {
		log.Println("Error retrieving photo info from mongoDB:", err)
		return nil, nil, err
	}

	file, err := os.Open(photoDB.FilePath)
	if err != nil {
		return nil, nil, err
	}

	return photoDB, file, nil
}

func (s *LocalPhotoStorage) GetPhotos(lastIdString string, limit int64) error {
	_, err := s.Db.GetPhotos(lastIdString, limit)
	if err != nil {
		log.Println("Error retrieving photos info from mongoDB:", err)
		return err
	}
	return nil
}

func (s *LocalPhotoStorage) SearchPhotosByLocation(long float64, lat float64, dist int) error {
	_, err := s.Db.SearchPhotosByLocation(long, lat, dist)
	if err != nil {
		log.Println("Error retrieving photos info from mongoDB:", err)
		return err
	}
	return nil
}
