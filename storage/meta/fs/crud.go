package fs

import (
	"github.com/podtserkovskiy/garnerd/storage"
)

type metaRW interface {
	read() (map[string]storage.Meta, error)
	write(data map[string]storage.Meta) error
	ping() error
}

type MetaCRUD struct {
	metaRW metaRW
}

func NewMetaCRUD(metaRW metaRW) *MetaCRUD {
	return &MetaCRUD{metaRW: metaRW}
}

func (s *MetaCRUD) Set(entry storage.Meta) error {
	data, err := s.metaRW.read()
	if err != nil {
		return err
	}

	data[entry.ImageName] = entry

	return s.metaRW.write(data)
}

func (s *MetaCRUD) Get(imageName string) (storage.Meta, error) {
	data, err := s.metaRW.read()
	if err != nil {
		return storage.Meta{}, err
	}

	entry, ok := data[imageName]
	if !ok {
		return storage.Meta{}, storage.ErrNotFound
	}

	return entry, nil
}

func (s *MetaCRUD) GetAll() ([]storage.Meta, error) {
	data, err := s.metaRW.read()
	if err != nil {
		return nil, err
	}

	list := make([]storage.Meta, 0, len(data))
	for _, v := range data {
		list = append(list, v)
	}

	return list, nil
}

func (s *MetaCRUD) Remove(imageName string) error {
	data, err := s.metaRW.read()
	if err != nil {
		return err
	}

	delete(data, imageName)

	return s.metaRW.write(data)
}

func (s *MetaCRUD) Ping() error {
	return s.metaRW.ping()
}
