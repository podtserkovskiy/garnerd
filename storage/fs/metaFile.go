package fs

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/docker/docker/pkg/ioutils"

	"github.com/podtserkovskiy/garnerd/storage"
)

type metaFile struct {
	mu   sync.Mutex
	path string
}

func newMetaFile(dir string) *metaFile {
	return &metaFile{path: filepath.Join(dir, "meta.json")}
}

func (f *metaFile) read() (map[string]storage.Meta, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	file, err := os.Open(f.path)

	switch {
	case os.IsNotExist(err):
		return map[string]storage.Meta{}, nil
	case err != nil:
		return nil, fmt.Errorf("cant open meta file, %w", err)
	}
	defer file.Close()

	data := map[string]storage.Meta{}
	if err := json.NewDecoder(file).Decode(&data); err != nil && err != io.EOF {
		return nil, fmt.Errorf("can't readFile meta file, %w", err)
	}

	return data, nil
}

func (f *metaFile) write(data map[string]storage.Meta) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	file, err := ioutils.NewAtomicFileWriter(f.path, 0666)
	if err != nil {
		return fmt.Errorf("can't create meta file writer, %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("can't writeFile meta file, %w", err)
	}

	return nil
}

func (f *metaFile) ping() error {
	dir := filepath.Dir(f.path)
	stat, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("ping '%s', %w", dir, err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("path '%s' is a file, directory is expected", dir) //nolint: goerr113
	}

	return nil
}
