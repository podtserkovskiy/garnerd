package fs

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/podtserkovskiy/garnerd/storage"
)

type metaCRUD interface {
	set(entry storage.Meta) error
	get(imageName string) (storage.Meta, error)
	remove(imageName string) error
	getAll() ([]storage.Meta, error)
	ping() error
}

type imgStorage interface {
	save(imgName string, imageDump io.Reader) error
	load(imgName string) (io.ReadCloser, error)
	remove(imgName string) error
	isExist(imageName string) (bool, error)
	removeNotIn(imageNames []string) error
	ping() error
}

type Storage struct {
	metaStorage metaCRUD
	imgStorage  imgStorage
}

func NewStorage(dir string) *Storage {
	return &Storage{metaStorage: newMetaFileCRUD(newMetaFile(dir)), imgStorage: newImgFileStorage(dir)}
}

func (s *Storage) Save(imageName, imageID string, imageDump io.Reader) error {
	if err := s.imgStorage.save(imageName, imageDump); err != nil {
		return err
	}

	err := s.metaStorage.set(storage.Meta{
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
	return s.imgStorage.load(imageName)
}

func (s *Storage) Remove(imageName string) error {
	err := s.metaStorage.remove(imageName)
	if err != nil {
		return err
	}

	return s.imgStorage.remove(imageName)
}

func (s *Storage) GetMeta(imageName string) (storage.Meta, error) {
	return s.metaStorage.get(imageName)
}

func (s *Storage) GetAllMeta() ([]storage.Meta, error) {
	return s.metaStorage.getAll()
}

func (s *Storage) Wait(ctx context.Context) error {
	fmt.Printf("Start waiting for storages")
	for {
		imgErr := s.imgStorage.ping()
		metaErr := s.metaStorage.ping()
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
	metas, err := s.metaStorage.getAll()
	if err != nil {
		return err
	}

	imageNames := make([]string, 0, len(metas))
	for _, meta := range metas {
		imageNames = append(imageNames, meta.ImageName)
	}

	err = s.imgStorage.removeNotIn(imageNames)
	if err != nil {
		return err
	}

	for _, meta := range metas {
		isExist, err := s.imgStorage.isExist(meta.ImageName)
		if err != nil {
			return err
		}

		if isExist {
			continue
		}

		if err := s.metaStorage.remove(meta.ImageName); err != nil {
			return err
		}
	}

	return nil
}
