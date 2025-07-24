package storage

import (
	"os"
	"photo-backup/model"
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

	_, err = file.WriteString(photo.FileContent)
	if err != nil {
		return err
	}

	return nil
}
