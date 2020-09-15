package separated

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/podtserkovskiy/garnerd/storage"
)

type MetaCRUD interface {
	Set(entry storage.Meta) error
	Get(imageName string) (storage.Meta, error)
	Remove(imageName string) error
	GetAll() ([]storage.Meta, error)
	Ping() error
}

type ImgStorage interface {
	Save(imgName string, imageDump io.Reader) error
	Load(imgName string) (io.ReadCloser, error)
	Remove(imgName string) error
	IsExist(imageName string) (bool, error)
	RemoveNotIn(imageNames []string) error
	Ping() error
}

type Storage struct {
	metaStorage MetaCRUD
	imgStorage  ImgStorage
}

func NewStorage(metaStorage MetaCRUD, imgStorage ImgStorage) *Storage {
	return &Storage{metaStorage: metaStorage, imgStorage: imgStorage}
}

func (s *Storage) Save(imageName, imageID string, imageDump io.Reader) error {
	if err := s.imgStorage.Save(imageName, imageDump); err != nil {
		return err
	}

	err := s.metaStorage.Set(storage.Meta{
		ImageName: imageName,
		ImageID:   imageID,
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("saving metadata, %w", err)
	}

	return nil
}

func (s *Storage) Load(imageName string) (io.ReadCloser, error) {
	return s.imgStorage.Load(imageName)
}

func (s *Storage) Remove(imageName string) error {
	err := s.metaStorage.Remove(imageName)
	if err != nil {
		return err
	}

	return s.imgStorage.Remove(imageName)
}

func (s *Storage) GetMeta(imageName string) (storage.Meta, error) {
	return s.metaStorage.Get(imageName)
}

func (s *Storage) GetAllMeta() ([]storage.Meta, error) {
	return s.metaStorage.GetAll()
}

func (s *Storage) Wait(ctx context.Context) error {
	fmt.Printf("Start waiting for storages")
	for {
		imgErr := s.imgStorage.Ping()
		metaErr := s.metaStorage.Ping()
		if imgErr == nil && metaErr == nil {
			fmt.Printf("\nStorages has been found successfully\n")

			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("storage is not ready, %w", ctx.Err())
		default:
			fmt.Print(".")
			time.Sleep(time.Second)
		}
	}
}

// CleanUp removes not paired images and metas.
func (s *Storage) CleanUp(ctx context.Context) error {
	metas, err := s.metaStorage.GetAll()
	if err != nil {
		return err
	}

	imageNames := make([]string, 0, len(metas))
	for _, meta := range metas {
		imageNames = append(imageNames, meta.ImageName)
	}

	err = s.imgStorage.RemoveNotIn(imageNames)
	if err != nil {
		return err
	}

	for _, meta := range metas {
		isExist, err := s.imgStorage.IsExist(meta.ImageName)
		if err != nil {
			return err
		}

		if isExist {
			continue
		}

		if err := s.metaStorage.Remove(meta.ImageName); err != nil {
			return err
		}
	}

	return nil
}
