package storage

import (
	"log"
	"os"
	"photo-backup/model"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

type PhotoStorage interface {
	SavePhoto(photo model.Photo) error
	GetPhoto(id string) (*model.Photo, error)
}

type LocalPhotoStorage struct {
	Directory string
	Db        PhotoDB
}

func (s *LocalPhotoStorage) SavePhoto(photo model.Photo) error {
	filePath := s.Directory + "/" + photo.Filename
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(photo.FileContent)
	if err != nil {
		return err
	}

	// reset file pointer to beginning for EXIF decoding
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}

	// extract exif data
	var geoPoint model.GeoPoint
	var takenAt time.Time
	x, err := exif.Decode(file)
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

	s.Db.SavePhoto(model.PhotoDB{
		Size:        photo.Size,
		ContentType: photo.ContentType,
		FilePath:    s.Directory + "/" + photo.Filename,
		TakenAt:     takenAt,
		LonLat:      geoPoint,
	})

	return nil
}

func (s *LocalPhotoStorage) GetPhoto(id string) (*model.Photo, error) {
	photoDB, err := s.Db.GetPhoto(id)
	if err != nil {
		log.Println("Error retrieving photo info from mongoDB:", err)
		return nil, err
	}

	file, err := os.Open(photoDB.FilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileContent := make([]byte, photoDB.Size)
	_, err = file.Read(fileContent)
	if err != nil {
		return nil, err
	}

	photo := &model.Photo{
		Size:        photoDB.Size,
		ContentType: photoDB.ContentType,
		Filename:    file.Name(),
		FileContent: fileContent,
	}
	return photo, nil
}
