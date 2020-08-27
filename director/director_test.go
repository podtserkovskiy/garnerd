// nolint: goerr113
package director

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/podtserkovskiy/garnerd/mocks"
	"github.com/podtserkovskiy/garnerd/storage"
)

func NewTestData() (*Director, *mocks.Cache, *mocks.Storage, *mocks.Docker, *mocks.Mover) {
	cache, storage, docker, mover := new(mocks.Cache), new(mocks.Storage), new(mocks.Docker), new(mocks.Mover)

	return NewDirector(cache, storage, docker, mover), cache, storage, docker, mover
}

func metaByUpdatedDesc() []storage.Meta {
	return []storage.Meta{
		{
			ImageName: "b-name",
			ImageID:   "b-id",
			UpdatedAt: time.Unix(2, 0),
		},
		{
			ImageName: "a-name",
			ImageID:   "a-id",
			UpdatedAt: time.Unix(1, 0),
		},
	}
}

func TestDirector_init(t *testing.T) {
	t.Run("returns an error when storage.GetAllMeta returns an error", func(t *testing.T) {
		director, _, sm, _, _ := NewTestData()
		sm.On("GetAllMeta").Return(nil, errors.New("storage err"))
		err := director.init(context.Background())
		require.EqualError(t, err, "getting persisted metadata, storage err")
	})

	t.Run("not updates cache when mover.FromStorageToDocker returns an error", func(t *testing.T) {
		director, _, sm, _, mm := NewTestData()
		sm.On("GetAllMeta").Return(metaByUpdatedDesc(), nil)
		mm.On("FromStorageToDocker", mock.Anything, mock.Anything).Return(errors.New("mover err"))
		err := director.init(context.Background())
		require.NoError(t, err)
	})

	t.Run("updates cache on successful moves", func(t *testing.T) {
		director, cm, sm, _, mm := NewTestData()
		sm.On("GetAllMeta").Return(metaByUpdatedDesc(), nil)
		mm.On("FromStorageToDocker", mock.Anything, mock.Anything).Return(nil)
		cm.On("AddSilent", "a-name", "a-id").Return().Once()
		cm.On("AddSilent", "b-name", "b-id").Return().Once()
		err := director.init(context.Background())
		require.NoError(t, err)
	})
}
