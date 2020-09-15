package fs

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func setUpTempDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal("can't create tempdir", err)
	}
	t.Cleanup(cleanUpTempDir(t, dir))

	return dir
}

func cleanUpTempDir(t *testing.T, dir string) func() {
	return func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Log("can't Remove tempdir", err)
		}
	}
}

func TestImgFileStorage_Ping(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		dir := setUpTempDir(t) + "/aaaa"
		storage := NewImgStorage(dir)
		err := storage.Ping()
		require.Error(t, err)
	})
	t.Run("is not a directory", func(t *testing.T) {
		dir := setUpTempDir(t) + "/aaaa"
		require.NoError(t, ioutil.WriteFile(dir, nil, 0600))
		storage := NewImgStorage(dir)
		err := storage.Ping()
		require.Error(t, err)
		require.Contains(t, err.Error(), "is a file, directory is expected")
	})
	t.Run("success", func(t *testing.T) {
		dir := setUpTempDir(t)
		storage := NewImgStorage(dir)
		err := storage.Ping()
		require.NoError(t, err)
	})
}
