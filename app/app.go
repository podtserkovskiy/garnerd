// nolint: goerr113
package app

import (
	"context"
	"fmt"

	"github.com/podtserkovskiy/garnerd/storage/compact"
	"github.com/podtserkovskiy/garnerd/storage/fs"
	"github.com/podtserkovskiy/garnerd/storage/separated"

	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"

	"github.com/podtserkovskiy/garnerd/cache/lru"
	"github.com/podtserkovskiy/garnerd/director"
	"github.com/podtserkovskiy/garnerd/docker"
	"github.com/podtserkovskiy/garnerd/mover"
)

func Start(maxCount int, dir string) error {
	dockerClient, err := client.NewEnvClient()
	if err != nil {
		return fmt.Errorf("can't create docker client, %s", err)
	}

	docker := docker.NewDaemon(dockerClient)

	ctx := context.Background()

	err = docker.Wait(ctx)
	if err != nil {
		return fmt.Errorf("waiting for docker daemon, %s", err)
	}

	log.Infof("Cache dir: %s", dir)
	storage := separated.NewStorage(fs.NewMetaFileCRUD(fs.NewMetaFile(dir)), compact.NewImgStorage(dir))
	err = storage.Wait(ctx)
	if err != nil {
		return fmt.Errorf("waiting for storage, %s", err)
	}

	err = storage.CleanUp(ctx)
	if err != nil {
		return fmt.Errorf("cleaning up, %s", err)
	}

	cache, err := lru.NewCache(maxCount)
	if err != nil {
		return fmt.Errorf("creating cache, %s", err)
	}

	director := director.NewDirector(cache, storage, docker, mover.NewMover(storage, docker))

	err = director.Start(ctx)
	if err != nil {
		return fmt.Errorf("start, %s", err)
	}

	return nil
}
