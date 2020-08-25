package mover

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/podtserkovskiy/garnerd/docker"
	"github.com/podtserkovskiy/garnerd/storage"
)

type Mover struct {
	storage storage.Storage
	docker  docker.Docker
}

func NewMover(storage storage.Storage, docker docker.Docker) *Mover {
	return &Mover{storage: storage, docker: docker}
}

func (m *Mover) FromDockerToStorage(ctx context.Context, imageName string) error {
	imageID, found, err := m.docker.ImageID(ctx, imageName)
	if err != nil {
		return fmt.Errorf("getting imageId, %w", err)
	}

	if !found {
		return fmt.Errorf("image has not been found in docker") // nolint: goerr113
	}

	dump, err := m.docker.SaveDump(ctx, imageName)
	if err != nil {
		return fmt.Errorf("dumping, %w", err)
	}
	defer dump.Close()

	err = m.storage.Save(imageName, imageID, dump)
	if err != nil {
		return fmt.Errorf("saving, %w", err)
	}

	return nil
}

func (m *Mover) FromStorageToDocker(ctx context.Context, imageName string) error {
	meta, err := m.storage.GetMeta(imageName)
	if err != nil {
		return fmt.Errorf("getting meta '%s' from storage, %w", imageName, err)
	}

	isSame, err := m.docker.ContainsSameVersion(ctx, meta.ImageName, meta.ImageID)
	if err != nil {
		return fmt.Errorf("checking '%s' in the daemon, %w", meta.ImageName, err)
	}

	if isSame {
		log.Infof("image '%s' already have been up to date", meta.ImageName)

		return nil
	}

	dump, err := m.storage.Load(meta.ImageName)
	if err != nil {
		return fmt.Errorf("loading '%s' from storage, %w", meta.ImageName, err)
	}
	defer dump.Close()

	err = m.docker.LoadDump(ctx, dump)
	if err != nil {
		return fmt.Errorf("loading '%s' into daemon, %w", meta.ImageName, err)
	}

	log.Infof("image '%s' has been successfully loaded", meta.ImageName)

	return nil
}
