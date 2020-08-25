// nolint: goerr113
package mover

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/podtserkovskiy/garnerd/mocks"
	"github.com/podtserkovskiy/garnerd/storage"
)

func NewTestData() (*Mover, *mocks.Storage, *mocks.Docker, context.Context) {
	storage, docker := new(mocks.Storage), new(mocks.Docker)

	return NewMover(storage, docker), storage, docker, context.Background()
}

func TestMover_FromDockerToStorage(t *testing.T) {
	t.Run("returns an error when docker.ImageID returns an error", func(t *testing.T) {
		mover, _, dm, ctx := NewTestData()
		dm.On("ImageID", ctx, "img-a").Return("", false, errors.New("docker error"))

		err := mover.FromDockerToStorage(ctx, "img-a")
		require.EqualError(t, err, "getting imageId, docker error")
	})

	t.Run("returns an error when docker image has not been found in docker", func(t *testing.T) {
		mover, _, dm, ctx := NewTestData()
		dm.On("ImageID", ctx, "img-a").Return("", false, nil)

		err := mover.FromDockerToStorage(ctx, "img-a")
		require.EqualError(t, err, "image has not been found in docker")
	})

	t.Run("returns an error when docker.SaveDump returns an error", func(t *testing.T) {
		mover, _, dm, ctx := NewTestData()
		dm.On("ImageID", ctx, "img-a").Return("id-a1", true, nil)
		dm.On("SaveDump", ctx, "img-a").Return(nil, errors.New("docker error"))

		err := mover.FromDockerToStorage(ctx, "img-a")
		require.EqualError(t, err, "dumping, docker error")
	})

	t.Run("returns an error when storage.Save returns an error", func(t *testing.T) {
		mover, sm, dm, ctx := NewTestData()
		dm.On("ImageID", ctx, "img-a").Return("id-a1", true, nil)
		file := ioutil.NopCloser(bytes.NewBufferString("aaa"))
		dm.On("SaveDump", ctx, "img-a").Return(file, nil)
		sm.On("Save", "img-a", "id-a1", mock.Anything).Return(errors.New("storage error"))

		err := mover.FromDockerToStorage(ctx, "img-a")
		require.EqualError(t, err, "saving, storage error")
	})

	t.Run("success", func(t *testing.T) {
		mover, sm, dm, ctx := NewTestData()
		dm.On("ImageID", ctx, "img-a").Return("id-a1", true, nil)
		file := ioutil.NopCloser(bytes.NewBufferString("aaa"))
		dm.On("SaveDump", ctx, "img-a").Return(file, nil)
		sm.On("Save", "img-a", "id-a1", mock.Anything).Return(nil)

		err := mover.FromDockerToStorage(ctx, "img-a")
		require.NoError(t, err)
	})
}

func TestMover_FromStorageToDocker(t *testing.T) {
	t.Run("returns an error when storage.GetMeta returns an error", func(t *testing.T) {
		mover, sm, _, ctx := NewTestData()
		sm.On("GetMeta", "img-a").Return(storage.Meta{}, errors.New("storage err"))

		err := mover.FromStorageToDocker(ctx, "img-a")
		require.EqualError(t, err, "getting meta 'img-a' from storage, storage err")
	})

	t.Run("returns an error when storage.GetMeta returns an error", func(t *testing.T) {
		mover, sm, dm, ctx := NewTestData()
		sm.On("GetMeta", "img-a").Return(storage.Meta{ImageName: "img-a", ImageID: "img-a1"}, nil)
		dm.On("ContainsSameVersion", ctx, "img-a", "img-a1").Return(false, errors.New("docker error"))

		err := mover.FromStorageToDocker(ctx, "img-a")
		require.EqualError(t, err, "checking 'img-a' in the daemon, docker error")
	})

	t.Run("success if images have the same docker id", func(t *testing.T) {
		mover, sm, dm, ctx := NewTestData()
		sm.On("GetMeta", "img-a").Return(storage.Meta{ImageName: "img-a", ImageID: "img-a1"}, nil)
		dm.On("ContainsSameVersion", ctx, "img-a", "img-a1").Return(true, nil)

		err := mover.FromStorageToDocker(ctx, "img-a")
		require.NoError(t, err, "checking 'img-a' in the daemon, docker error")
	})

	t.Run("returns an error when storage.Load returns an error", func(t *testing.T) {
		mover, sm, dm, ctx := NewTestData()
		sm.On("GetMeta", "img-a").Return(storage.Meta{ImageName: "img-a", ImageID: "img-a1"}, nil)
		dm.On("ContainsSameVersion", ctx, "img-a", "img-a1").Return(false, nil)
		sm.On("Load", "img-a").Return(nil, errors.New("storage error"))

		err := mover.FromStorageToDocker(ctx, "img-a")
		require.EqualError(t, err, "loading 'img-a' from storage, storage error")
	})

	t.Run("returns an error when docker.LoadDump returns an error", func(t *testing.T) {
		mover, sm, dm, ctx := NewTestData()
		sm.On("GetMeta", "img-a").Return(storage.Meta{ImageName: "img-a", ImageID: "img-a1"}, nil)
		dm.On("ContainsSameVersion", ctx, "img-a", "img-a1").Return(false, nil)
		file := ioutil.NopCloser(bytes.NewBufferString("aaa"))
		sm.On("Load", "img-a").Return(file, nil)
		dm.On("LoadDump", ctx, mock.Anything).Return(errors.New("docker error"))

		err := mover.FromStorageToDocker(ctx, "img-a")
		require.EqualError(t, err, "loading 'img-a' into daemon, docker error")
	})

	t.Run("successfully loaded", func(t *testing.T) {
		mover, sm, dm, ctx := NewTestData()
		sm.On("GetMeta", "img-a").Return(storage.Meta{ImageName: "img-a", ImageID: "img-a1"}, nil)
		dm.On("ContainsSameVersion", ctx, "img-a", "img-a1").Return(false, nil)
		file := ioutil.NopCloser(bytes.NewBufferString("aaa"))
		sm.On("Load", "img-a").Return(file, nil)
		dm.On("LoadDump", ctx, mock.Anything).Return(nil)

		err := mover.FromStorageToDocker(ctx, "img-a")
		require.NoError(t, err)
	})
}
