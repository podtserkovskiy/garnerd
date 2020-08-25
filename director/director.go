package director

import (
	"context"
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/podtserkovskiy/garnerd/docker"
	"github.com/podtserkovskiy/garnerd/storage"
)

type Cache interface {
	AddSilent(imageName, imageID string)
	Add(imageName, imageID string)
	OnAdd(func(imageName, imageID string))
	OnEvict(func(imageName, imageID string))
}

type Mover interface {
	FromDockerToStorage(ctx context.Context, imageName string) error
	FromStorageToDocker(ctx context.Context, imageName string) error
}

type Director struct {
	cache   Cache
	mover   Mover
	storage storage.Storage
	docker  docker.Docker
}

func NewDirector(cache Cache, storage storage.Storage, docker docker.Docker, mover Mover) *Director {
	return &Director{cache: cache, storage: storage, docker: docker, mover: mover}
}

func (d *Director) Start(ctx context.Context) error {
	d.cache.OnAdd(d.saveImg(ctx))
	d.cache.OnEvict(d.removeImg())

	if err := d.init(ctx); err != nil {
		return fmt.Errorf("init, %w", err)
	}
	d.listenContainerCreated(ctx) // until docker sends events

	return nil
}

func (d *Director) init(ctx context.Context) error {
	metas, err := d.storage.GetAllMeta()
	if err != nil {
		return fmt.Errorf("getting persisted metadata, %w", err)
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].UpdatedAt.UnixNano() < metas[j].UpdatedAt.UnixNano()
	})

	for _, meta := range metas {
		if err := d.mover.FromStorageToDocker(ctx, meta.ImageName); err != nil {
			log.Errorf("loading '%s' from storage, %s", meta.ImageName, err)

			continue
		}

		d.cache.AddSilent(meta.ImageName, meta.ImageID)
	}

	return nil
}

func (d *Director) saveImg(ctx context.Context) func(imageName, imageID string) {
	return func(imageName, imageID string) {
		if err := d.mover.FromDockerToStorage(ctx, imageName); err != nil {
			log.Warnf("Caching '%s', %s", imageName, err)

			return
		}
		log.Infof("Image '%s' has been cached", imageName)
	}
}

func (d *Director) removeImg() func(imageName, imageID string) {
	return func(imageName, imageID string) {
		if err := d.storage.Remove(imageName); err != nil {
			log.Warnf("Removing '%s', %s", imageName, err)

			return
		}
		log.Infof("Image '%s' has been evicted", imageName)
	}
}

func (d *Director) listenContainerCreated(ctx context.Context) {
	log.Info("Listening for new containers")
	for container := range d.docker.ListenContainerCreation(ctx) {
		d.cache.Add(container.ImageName, container.ImageID)
	}
}
