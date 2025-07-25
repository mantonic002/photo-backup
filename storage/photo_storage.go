package storage

import (
	"log"
	"os"
	"photo-backup/model"

	"github.com/rwcarlsen/goexif/exif"
)

type PhotoStorage interface {
	SavePhoto(photo model.Photo) error
}

type LocalPhotoStorage struct {
	Directory string
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

	x, err := exif.Decode(file)
	if err != nil {
		log.Println("Error decoding EXIF data, proceeding without it:", err)
		return nil
	}

	tm, err := x.DateTime()
	if err != nil {
		log.Println("No datetime in EXIF data:", err)
	} else {
		log.Println("Taken:", tm)
	}

	lat, long, err := x.LatLong()
	if err != nil {
		log.Println("No lat/long in EXIF data:", err)
	} else {
		log.Println("lat, long:", lat, ",", long)
	}

	return nil
}
