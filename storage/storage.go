package storage

import (
	"errors"
	"io"
	"time"
)

type Meta struct {
	// repo.com/aaa/bbb:tag
	ImageName string
	// docker image id
	ImageID   string
	UpdatedAt time.Time
}

var ErrNotFound = errors.New("not found")

type Storage interface {
	Save(imageName, imageID string, imageDump io.Reader) error
	Load(imageName string) (io.ReadCloser, error)
	Remove(imageName string) error
	GetMeta(imageName string) (Meta, error)
	GetAllMeta() ([]Meta, error)
}
