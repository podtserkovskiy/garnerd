package fs

import (
	"github.com/podtserkovskiy/garnerd/storage"
)

type metaRW interface {
	read() (map[string]storage.Meta, error)
	write(data map[string]storage.Meta) error
	ping() error
}

type MetaFileCRUD struct {
	metaRW metaRW
}

func NewMetaFileCRUD(metaRW metaRW) *MetaFileCRUD {
	return &MetaFileCRUD{metaRW: metaRW}
}

func (s *MetaFileCRUD) Set(entry storage.Meta) error {
	data, err := s.metaRW.read()
	if err != nil {
		return err
	}

	data[entry.ImageName] = entry

	return s.metaRW.write(data)
}

func (s *MetaFileCRUD) Get(imageName string) (storage.Meta, error) {
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

func (s *MetaFileCRUD) GetAll() ([]storage.Meta, error) {
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

func (s *MetaFileCRUD) Remove(imageName string) error {
	data, err := s.metaRW.read()
	if err != nil {
		return err
	}

	delete(data, imageName)

	return s.metaRW.write(data)
}

func (s *MetaFileCRUD) Ping() error {
	return s.metaRW.ping()
}
