// nolint: goerr113,funlen
package separated

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/podtserkovskiy/garnerd/storage"
)

type metaCRUDMock struct {
	mock.Mock
}

func (m *metaCRUDMock) Set(entry storage.Meta) error {
	args := m.Called(entry)

	return args.Error(0)
}

func (m *metaCRUDMock) Get(imageName string) (storage.Meta, error) {
	args := m.Called(imageName)

	return args.Get(0).(storage.Meta), args.Error(1)
}

func (m *metaCRUDMock) Remove(imageName string) error {
	args := m.Called(imageName)

	return args.Error(0)
}

func (m *metaCRUDMock) GetAll() ([]storage.Meta, error) {
	args := m.Called()
	metas, _ := args.Get(0).([]storage.Meta)

	return metas, args.Error(1)
}

func (m *metaCRUDMock) Ping() error {
	args := m.Called()

	return args.Error(0)
}

type imgStorageMock struct {
	mock.Mock
}

func (m *imgStorageMock) Save(imgName string, imageDump io.Reader) error {
	args := m.Called(imgName, imageDump)

	return args.Error(0)
}

func (m *imgStorageMock) Load(imgName string) (io.ReadCloser, error) {
	args := m.Called(imgName)
	rc, _ := args.Get(0).(io.ReadCloser)

	return rc, args.Error(1)
}

func (m *imgStorageMock) Remove(imgName string) error {
	args := m.Called(imgName)

	return args.Error(0)
}

func (m *imgStorageMock) IsExist(imageName string) (bool, error) {
	args := m.Called(imageName)

	return args.Bool(0), args.Error(1)
}

func (m *imgStorageMock) RemoveNotIn(imageNames []string) error {
	args := m.Called(imageNames)

	return args.Error(0)
}

func (m *imgStorageMock) Ping() error {
	args := m.Called()

	return args.Error(0)
}

func TestStorage_CleanUp(t *testing.T) {
	t.Run("MetaCRUD.GetAll returns error", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		metaCRUD.On("GetAll").Return(nil, errors.New("some err"))

		stor := &Storage{metaStorage: metaCRUD, imgStorage: nil}
		err := stor.CleanUp(context.Background())
		require.EqualError(t, err, "some err")
	})

	t.Run("ImgStorage.RemoveNotIn receives correct images list ", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		imgStorage := &imgStorageMock{}
		metaCRUD.On("GetAll").Return([]storage.Meta{{ImageName: "a"}, {ImageName: "b"}}, nil)
		imgStorage.On("RemoveNotIn", []string{"a", "b"}).Return(errors.New("some err"))

		stor := &Storage{metaStorage: metaCRUD, imgStorage: imgStorage}
		_ = stor.CleanUp(context.Background())
	})

	t.Run("ImgStorage.RemoveNotIn returns an error", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		imgStorage := &imgStorageMock{}
		metaCRUD.On("GetAll").Return(nil, nil)
		imgStorage.On("RemoveNotIn", mock.Anything).Return(errors.New("some err"))

		stor := &Storage{metaStorage: metaCRUD, imgStorage: imgStorage}
		err := stor.CleanUp(context.Background())
		require.EqualError(t, err, "some err")
	})

	t.Run("ImgStorage.IsExist returns an error", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		imgStorage := &imgStorageMock{}
		metaCRUD.On("GetAll").Return([]storage.Meta{{ImageName: "a"}}, nil)
		imgStorage.On("RemoveNotIn", mock.Anything).Return(nil)
		imgStorage.On("IsExist", mock.Anything).Return(false, errors.New("some err"))

		stor := &Storage{metaStorage: metaCRUD, imgStorage: imgStorage}
		err := stor.CleanUp(context.Background())
		require.EqualError(t, err, "some err")
	})

	t.Run("ImgStorage.Remove returns an error", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		imgStorage := &imgStorageMock{}
		metaCRUD.On("GetAll").Return([]storage.Meta{{ImageName: "a"}}, nil)
		imgStorage.On("RemoveNotIn", mock.Anything).Return(nil)
		imgStorage.On("IsExist", mock.Anything).Return(false, nil)
		metaCRUD.On("Remove", mock.Anything).Return(errors.New("some err"))

		stor := &Storage{metaStorage: metaCRUD, imgStorage: imgStorage}
		err := stor.CleanUp(context.Background())
		require.EqualError(t, err, "some err")
	})

	t.Run("image has been deleted", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		imgStorage := &imgStorageMock{}
		metaCRUD.On("GetAll").Return([]storage.Meta{{ImageName: "a"}}, nil)
		imgStorage.On("RemoveNotIn", mock.Anything).Return(nil)
		imgStorage.On("IsExist", mock.Anything).Return(false, nil)
		metaCRUD.On("Remove", "a").Return(nil)

		stor := &Storage{metaStorage: metaCRUD, imgStorage: imgStorage}
		err := stor.CleanUp(context.Background())
		require.NoError(t, err)
	})

	t.Run("image has not been deleted", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		imgStorage := &imgStorageMock{}
		metaCRUD.On("GetAll").Return([]storage.Meta{{ImageName: "a"}}, nil)
		imgStorage.On("RemoveNotIn", mock.Anything).Return(nil)
		imgStorage.On("IsExist", mock.Anything).Return(true, nil)

		stor := &Storage{metaStorage: metaCRUD, imgStorage: imgStorage}
		err := stor.CleanUp(context.Background())
		require.NoError(t, err)
	})
}

func TestStorage_Wait(t *testing.T) {
	t.Run("context has error", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		imgStorage := &imgStorageMock{}
		metaCRUD.On("Ping").Return(errors.New("some 1")).Once()
		imgStorage.On("Ping").Return(errors.New("some 2")).Once()

		stor := &Storage{metaStorage: metaCRUD, imgStorage: imgStorage}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := stor.Wait(ctx)
		require.EqualError(t, err, "storage is not ready, context canceled")
	})

	t.Run("storage available after 1 try", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		imgStorage := &imgStorageMock{}

		// first try
		metaCRUD.On("Ping").Return(errors.New("some 1")).Once()
		imgStorage.On("Ping").Return(errors.New("some 2")).Once()

		//// second try
		metaCRUD.On("Ping").Return(nil).Once()
		imgStorage.On("Ping").Return(nil).Once()

		stor := &Storage{metaStorage: metaCRUD, imgStorage: imgStorage}
		err := stor.Wait(context.Background())
		require.NoError(t, err)
	})
}

func TestStorage_GetAllMeta(t *testing.T) {
	t.Run("storage returns an error", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		metaCRUD.On("GetAll").Return(nil, errors.New("some err"))
		stor := &Storage{metaStorage: metaCRUD, imgStorage: nil}
		metas, err := stor.GetAllMeta()
		require.Error(t, err, "")
		require.Empty(t, metas)
	})

	t.Run("success", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		metaCRUD.On("GetAll").Return([]storage.Meta{{}, {}}, nil)
		stor := &Storage{metaStorage: metaCRUD, imgStorage: nil}
		metas, err := stor.GetAllMeta()
		require.NoError(t, err)
		require.Len(t, metas, 2)
	})
}

func TestStorage_Load(t *testing.T) {
	t.Run("storage returns an error", func(t *testing.T) {
		imgStorage := &imgStorageMock{}
		imgStorage.On("Load", mock.Anything).Return(nil, errors.New("some err"))
		stor := &Storage{metaStorage: nil, imgStorage: imgStorage}
		_, err := stor.Load("")
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		imgStorage := &imgStorageMock{}
		imgStorage.On("Load", mock.Anything).Return(nil, nil)
		stor := &Storage{metaStorage: nil, imgStorage: imgStorage}
		_, err := stor.Load("")
		require.NoError(t, err)
	})
}

func TestStorage_GetMeta(t *testing.T) {
	t.Run("storage returns an error", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		metaCRUD.On("Get", mock.Anything).Return(storage.Meta{}, errors.New("some err"))
		stor := &Storage{metaStorage: metaCRUD, imgStorage: nil}
		_, err := stor.GetMeta("")
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		metaCRUD.On("Get", mock.Anything).Return(storage.Meta{}, nil)
		stor := &Storage{metaStorage: metaCRUD, imgStorage: nil}
		_, err := stor.GetMeta("")
		require.NoError(t, err)
	})
}

func TestStorage_Remove(t *testing.T) {
	t.Run("meta returns an error", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		metaCRUD.On("Remove", mock.Anything).Return(errors.New("meta err"))
		imgStorage := &imgStorageMock{}
		stor := &Storage{metaStorage: metaCRUD, imgStorage: imgStorage}
		err := stor.Remove("")
		require.EqualError(t, err, "meta err")
	})

	t.Run("img returns an error", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		metaCRUD.On("Remove", mock.Anything).Return(nil)
		imgStorage := &imgStorageMock{}
		imgStorage.On("Remove", mock.Anything).Return(errors.New("img err"))
		stor := &Storage{metaStorage: metaCRUD, imgStorage: imgStorage}
		err := stor.Remove("")
		require.EqualError(t, err, "img err")
	})

	t.Run("success", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		metaCRUD.On("Remove", mock.Anything).Return(nil)
		imgStorage := &imgStorageMock{}
		imgStorage.On("Remove", mock.Anything).Return(nil)
		stor := &Storage{metaStorage: metaCRUD, imgStorage: imgStorage}
		err := stor.Remove("")
		require.NoError(t, err)
	})
}

func TestStorage_Save(t *testing.T) {
	t.Run("img returns an error", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		imgStorage := &imgStorageMock{}
		reader := bytes.NewBufferString("cc")
		imgStorage.On("Save", "aa", reader).Return(errors.New("img err"))

		stor := &Storage{metaStorage: metaCRUD, imgStorage: imgStorage}
		err := stor.Save("aa", "bb", reader)
		require.EqualError(t, err, "img err")
	})

	t.Run("meta returns an error", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		imgStorage := &imgStorageMock{}
		reader := bytes.NewBufferString("cc")
		imgStorage.On("Save", "aa", reader).Return(nil)
		metaCRUD.On("Set", mock.Anything).Return(errors.New("meta err"))

		stor := &Storage{metaStorage: metaCRUD, imgStorage: imgStorage}
		err := stor.Save("aa", "bb", reader)
		require.EqualError(t, err, "saving metadata, meta err")
	})

	t.Run("success", func(t *testing.T) {
		metaCRUD := &metaCRUDMock{}
		imgStorage := &imgStorageMock{}
		reader := bytes.NewBufferString("cc")
		imgStorage.On("Save", "aa", reader).Return(nil)
		metaCRUD.On("Set", mock.Anything).Return(nil)

		stor := &Storage{metaStorage: metaCRUD, imgStorage: imgStorage}
		err := stor.Save("aa", "bb", reader)
		require.NoError(t, err)
	})
}
