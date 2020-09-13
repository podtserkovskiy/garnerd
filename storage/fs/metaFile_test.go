package fs

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/podtserkovskiy/garnerd/storage"
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

func readMetaFile(t *testing.T, dir string) string {
	data, err := ioutil.ReadFile(path.Join(dir, "meta.json"))
	if err != nil {
		t.Fatal("can't readFile metafile", err)
	}

	return string(data)
}

func writeMetaFile(t *testing.T, dir, content string) {
	err := ioutil.WriteFile(path.Join(dir, "meta.json"), []byte(content), 0600)
	if err != nil {
		t.Fatal("can't writeFile metafile", err)
	}
}

func noFile(t *testing.T, dir string) {}

func emptyFile(t *testing.T, dir string) {
	content := ""
	writeMetaFile(t, dir, content)
}

func emptyJSON(t *testing.T, dir string) {
	content := "{}"
	writeMetaFile(t, dir, content)
}

func invalidJSON(t *testing.T, dir string) {
	content := "}{"
	writeMetaFile(t, dir, content)
}

func twoEntries(t *testing.T, dir string) {
	content := `{
		"ubuntu:1.0": {"ImageID": "hash:1111", "ImageName":"ubuntu:1.0", "UpdatedAt":"1970-01-01T03:00:23+03:00"},
		"debian:2.0": {"ImageID": "hash:2222", "ImageName":"debian:2.0", "UpdatedAt":"1970-01-01T03:00:24+03:00"}
	}`
	writeMetaFile(t, dir, content)
}

func TestMetaFile_WriteSuccess(t *testing.T) {
	cases := []struct {
		name        string
		makeFixture func(t *testing.T, dir string)
		expData     map[string]storage.Meta
	}{
		{
			name:        "noFile",
			makeFixture: noFile,
			expData:     map[string]storage.Meta{},
		},
		{
			name:        "emptyFile",
			makeFixture: emptyFile,
			expData:     map[string]storage.Meta{},
		},
		{
			name:        "emptyJSON",
			makeFixture: emptyJSON,
			expData:     map[string]storage.Meta{},
		},
		{
			name:        "twoEntries",
			makeFixture: twoEntries,
			expData: map[string]storage.Meta{
				"ubuntu:1.0": {ImageID: "hash:1111", ImageName: "ubuntu:1.0", UpdatedAt: time.Unix(23, 0)},
				"debian:2.0": {ImageID: "hash:2222", ImageName: "debian:2.0", UpdatedAt: time.Unix(24, 0)},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := setUpTempDir(t)
			tc.makeFixture(t, dir)
			file := NewMetaFile(dir)
			data, err := file.read()
			require.NoError(t, err)
			require.Equal(t, tc.expData, data)
		})
	}
}

func TestMetaFile_WriteError(t *testing.T) {
	cases := []struct {
		name        string
		makeFixture func(t *testing.T, dir string)
	}{
		{
			name:        "invalidJSON",
			makeFixture: invalidJSON,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := setUpTempDir(t)
			tc.makeFixture(t, dir)
			file := NewMetaFile(dir)
			_, err := file.read()
			require.EqualError(t, err, "can't readFile meta file, invalid character '}' looking for beginning of value")
		})
	}
}

func TestMetaFile_Write(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir := setUpTempDir(t)
		file := NewMetaFile(dir)
		data := map[string]storage.Meta{
			"ubuntu:1.0": {ImageID: "hash:1111", ImageName: "ubuntu:1.0", UpdatedAt: time.Unix(23, 0)},
			"debian:2.0": {ImageID: "hash:2222", ImageName: "debian:2.0", UpdatedAt: time.Unix(24, 0)},
		}
		err := file.write(data)
		require.NoError(t, err)
		fileContent := readMetaFile(t, dir)
		expectedFileContent := `{
			"ubuntu:1.0": {"ImageID": "hash:1111", "ImageName":"ubuntu:1.0", "UpdatedAt":"1970-01-01T03:00:23+03:00"},
			"debian:2.0": {"ImageID": "hash:2222", "ImageName":"debian:2.0", "UpdatedAt":"1970-01-01T03:00:24+03:00"}
		}`
		require.JSONEq(t, fileContent, expectedFileContent)
	})
}

// nolint: dupl
func TestMetaFile_Ping(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		dir := setUpTempDir(t) + "/aaaa"
		file := NewMetaFile(dir)
		err := file.ping()
		require.Error(t, err)
	})
	t.Run("is not a directory", func(t *testing.T) {
		dir := setUpTempDir(t) + "/aaaa"
		require.NoError(t, ioutil.WriteFile(dir, nil, 0600))
		file := NewMetaFile(dir)
		err := file.ping()
		require.Error(t, err)
		require.Contains(t, err.Error(), "is a file, directory is expected")
	})
	t.Run("success", func(t *testing.T) {
		dir := setUpTempDir(t)
		file := NewMetaFile(dir)
		err := file.ping()
		require.NoError(t, err)
	})
}
