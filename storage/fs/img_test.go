package fs

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

// nolint: dupl
func TestImgFileStorage_Ping(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		dir := setUpTempDir(t) + "/aaaa"
		storage := newImgFileStorage(dir)
		err := storage.ping()
		require.Error(t, err)
	})
	t.Run("is not a directory", func(t *testing.T) {
		dir := setUpTempDir(t) + "/aaaa"
		require.NoError(t, ioutil.WriteFile(dir, nil, 0600))
		storage := newImgFileStorage(dir)
		err := storage.ping()
		require.Error(t, err)
		require.Contains(t, err.Error(), "is a file, directory is expected")
	})
	t.Run("success", func(t *testing.T) {
		dir := setUpTempDir(t)
		storage := newImgFileStorage(dir)
		err := storage.ping()
		require.NoError(t, err)
	})
}
