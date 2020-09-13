// nolint: goerr113
package fs

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/podtserkovskiy/garnerd/storage"
)

type metaRWMock struct {
	mock.Mock
}

func (m *metaRWMock) read() (map[string]storage.Meta, error) {
	args := m.Called()
	meta, _ := args.Get(0).(map[string]storage.Meta)

	return meta, args.Error(1)
}

func (m *metaRWMock) write(data map[string]storage.Meta) error {
	args := m.Called(data)

	return args.Error(0)
}

func (m *metaRWMock) ping() error {
	panic("implement")
}

func TestMeta_Set(t *testing.T) {
	t.Run("can't read prev version", func(t *testing.T) {
		metaRW := &metaRWMock{}
		metaRW.On("read").Return(nil, errors.New("somerr"))

		metaCRUD := NewMetaFileCRUD(metaRW)
		err := metaCRUD.Set(storage.Meta{})
		require.EqualError(t, err, "somerr")
	})

	t.Run("success", func(t *testing.T) {
		metaRW := &metaRWMock{}
		metaRW.On("read").Return(map[string]storage.Meta{}, nil)

		expectedMetaEntry := storage.Meta{ImageName: "aa", ImageID: "bb", UpdatedAt: time.Time{}}
		metaRW.On("write", map[string]storage.Meta{"aa": expectedMetaEntry}).Return(nil)

		metaCRUD := NewMetaFileCRUD(metaRW)
		err := metaCRUD.Set(expectedMetaEntry)
		require.NoError(t, err)
	})
}

func TestMeta_Get(t *testing.T) {
	t.Run("can't read prev version", func(t *testing.T) {
		metaRW := &metaRWMock{}
		metaRW.On("read").Return(nil, errors.New("somerr"))

		metaCRUD := NewMetaFileCRUD(metaRW)
		_, err := metaCRUD.Get("aaa:123")
		require.EqualError(t, err, "somerr")
	})

	t.Run("not found", func(t *testing.T) {
		metaRW := &metaRWMock{}
		metaRW.On("read").Return(map[string]storage.Meta{}, nil)

		metaCRUD := NewMetaFileCRUD(metaRW)
		_, err := metaCRUD.Get("aaa:123")
		require.Equal(t, err, storage.ErrNotFound)
	})

	t.Run("success", func(t *testing.T) {
		metaRW := &metaRWMock{}

		expMetaEntry1 := storage.Meta{ImageName: "aaa:123", ImageID: "bb", UpdatedAt: time.Time{}}
		expMetaEntry2 := storage.Meta{ImageName: "bbb:123", ImageID: "bb", UpdatedAt: time.Time{}}
		readReturns := map[string]storage.Meta{"aaa:123": expMetaEntry1, "bbb:123": expMetaEntry2}
		metaRW.On("read").Return(readReturns, nil)

		metaCRUD := NewMetaFileCRUD(metaRW)
		_, err := metaCRUD.Get("aaa:123")
		require.NoError(t, err)
	})
}

func TestMeta_GetAll(t *testing.T) {
	t.Run("can't read prev version", func(t *testing.T) {
		metaRW := &metaRWMock{}
		metaRW.On("read").Return(nil, errors.New("somerr"))

		metaCRUD := NewMetaFileCRUD(metaRW)
		_, err := metaCRUD.GetAll()
		require.EqualError(t, err, "somerr")
	})

	t.Run("success", func(t *testing.T) {
		metaRW := &metaRWMock{}

		expMetaEntry1 := storage.Meta{ImageName: "aaa:123", ImageID: "bb", UpdatedAt: time.Time{}}
		expMetaEntry2 := storage.Meta{ImageName: "bbb:123", ImageID: "bb", UpdatedAt: time.Time{}}
		readReturns := map[string]storage.Meta{"aaa:123": expMetaEntry1, "bbb:123": expMetaEntry2}
		metaRW.On("read").Return(readReturns, nil)

		metaCRUD := NewMetaFileCRUD(metaRW)
		all, err := metaCRUD.GetAll()
		require.NoError(t, err)
		require.ElementsMatch(t, all, []storage.Meta{expMetaEntry1, expMetaEntry2})
	})
}

func TestMeta_Remove(t *testing.T) {
	t.Run("can't read prev version", func(t *testing.T) {
		metaRW := &metaRWMock{}
		metaRW.On("read").Return(nil, errors.New("somerr"))

		metaCRUD := NewMetaFileCRUD(metaRW)
		err := metaCRUD.Remove("aaa:111")
		require.EqualError(t, err, "somerr")
	})

	t.Run("success", func(t *testing.T) {
		metaRW := &metaRWMock{}
		metaEntry1 := storage.Meta{ImageName: "aaa:111", ImageID: "bb", UpdatedAt: time.Time{}}
		metaEntry2 := storage.Meta{ImageName: "aaa:222", ImageID: "bb", UpdatedAt: time.Time{}}
		readReturns := map[string]storage.Meta{"aaa:111": metaEntry1, "aaa:222": metaEntry2}
		metaRW.On("read").Return(readReturns, nil)

		writeArg := map[string]storage.Meta{"aaa:222": metaEntry2}
		metaRW.On("write", writeArg).Return(nil)

		metaCRUD := NewMetaFileCRUD(metaRW)
		err := metaCRUD.Remove("aaa:111")
		require.NoError(t, err)
	})
}
