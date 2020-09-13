package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

type Docker interface {
	Wait(ctx context.Context) error
	SaveDump(ctx context.Context, name string) (io.ReadCloser, error)
	LoadDump(ctx context.Context, image io.Reader) error
	ListenContainerCreation(ctx context.Context) <-chan ContainerCreated
	ContainsSameVersion(ctx context.Context, yourImageID, imageName string) (bool, error)
	ImageID(ctx context.Context, imageName string) (string, bool, error)
}

type ContainerCreated struct {
	ImageID   string
	ImageName string
}

type Daemon struct {
	client *client.Client
}

func NewDaemon(client *client.Client) *Daemon {
	return &Daemon{client: client}
}

func (w *Daemon) ListenContainerCreation(ctx context.Context) <-chan ContainerCreated {
	filterArgs := filters.NewArgs()
	filterArgs.Add("type", events.ImageEventType)
	filterArgs.Add("event", "pull")
	filter := types.EventsOptions{
		Filters: filterArgs,
	}
	msgs, errs := w.client.Events(ctx, filter)

	go func() {
		for err := range errs {
			fmt.Println("Event error: ", err)
		}
	}()

	resChan := make(chan ContainerCreated)
	go func() {
		for msg := range msgs {
			imageName := msg.Actor.ID
			log.Infof("Image '%s' has been used", imageName)
			inspect, _, err := w.client.ImageInspectWithRaw(ctx, imageName)
			if err != nil {
				log.Warn("inspect error,", err)

				continue
			}

			resChan <- ContainerCreated{
				ImageID:   inspect.ID,
				ImageName: imageName,
			}
		}
		close(resChan)
	}()

	return resChan
}

func (w *Daemon) SaveDump(ctx context.Context, name string) (io.ReadCloser, error) {
	image, err := w.client.ImageSave(ctx, []string{name})
	if err != nil {
		return nil, err
	}

	return image, nil
}

func (w *Daemon) LoadDump(ctx context.Context, image io.Reader) error {
	resp, err := w.client.ImageLoad(ctx, image, false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.JSON {
		jsonError := struct {
			Error string `json:"error"`
		}{}

		if err = json.NewDecoder(resp.Body).Decode(&jsonError); err != nil {
			return err
		}

		if len(jsonError.Error) > 0 {
			return errors.New(jsonError.Error) // nolint: goerr113
		}
	}

	return nil
}

// ImageID returns ImageID, isFound and err.
func (w *Daemon) ImageID(ctx context.Context, imageName string) (string, bool, error) {
	inspect, _, err := w.client.ImageInspectWithRaw(ctx, imageName)
	if client.IsErrNotFound(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("getting info from docker, %w", err)
	}

	return inspect.ID, true, nil
}

func (w *Daemon) ContainsSameVersion(ctx context.Context, imageName, yourImageID string) (bool, error) {
	inspect, _, err := w.client.ImageInspectWithRaw(ctx, imageName)
	if client.IsErrNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("getting info from docker, %w", err)
	}

	return inspect.ID == yourImageID, nil
}

func (w *Daemon) Wait(ctx context.Context) error {
	log.Println("Waiting for docker ")
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("docker is not ready, %w", ctx.Err())
		default:
		}

		if _, err := w.client.Ping(ctx); err != nil {
			fmt.Print(".")
			time.Sleep(time.Second)

			continue
		}

		break
	}
	fmt.Println("\nDocker has been found")

	return nil
}
