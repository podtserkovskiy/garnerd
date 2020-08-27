package fs

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/docker/docker/pkg/ioutils"
)

var (
	imageSlug    = regexp.MustCompile(`[^\w\d]`)        // nolint: gochecknoglobals
	tmpCacheFile = regexp.MustCompile(`\..+\.cache\d?`) // nolint: gochecknoglobals
)

type imgFileStorage struct {
	dir string
}

func newImgFileStorage(dir string) *imgFileStorage {
	return &imgFileStorage{dir: dir}
}

func (i *imgFileStorage) save(imgName string, imageDump io.Reader) error {
	imagePath := i.imagePath(imgName)
	file, err := ioutils.NewAtomicFileWriter(imagePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("creating file '%s', %w", imagePath, err)
	}
	defer file.Close()

	if _, err = io.Copy(file, imageDump); err != nil {
		return fmt.Errorf("copying the dump to '%s', %w", imagePath, err)
	}

	return nil
}

func (i *imgFileStorage) load(imgName string) (io.ReadCloser, error) {
	imagePath := i.imagePath(imgName)
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("can't open '%s', %w", imagePath, err)
	}

	return file, nil
}

func (i *imgFileStorage) remove(imgName string) error {
	imagePath := i.imagePath(imgName)
	err := os.RemoveAll(imagePath)
	if err != nil {
		return fmt.Errorf("can't remove '%s', %w", imagePath, err)
	}

	return nil
}

func (i *imgFileStorage) isExist(imageName string) (bool, error) {
	_, err := os.Stat(i.imagePath(imageName))
	switch {
	case os.IsNotExist(err):
		return false, nil
	case err == nil:
		return true, nil
	}

	return false, fmt.Errorf("checking existence '%s', %w", imageName, err)
}

func (i *imgFileStorage) removeNotIn(imageNames []string) error {
	allowedSet := map[string]bool{}
	for _, name := range imageNames {
		allowedSet[i.imagePath(name)] = true
	}

	return filepath.Walk(i.dir, func(path string, info os.FileInfo, err error) error {
		if tmpCacheFile.MatchString(filepath.Base(path)) {
			return os.RemoveAll(path)
		}
		if filepath.Ext(path) == ".cache" && !allowedSet[path] {
			return os.RemoveAll(path)
		}

		return nil
	})
}

func (i *imgFileStorage) ping() error {
	stat, err := os.Stat(i.dir)
	if err != nil {
		return fmt.Errorf("ping '%s', %w", i.dir, err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("path '%s' is a file, directory is expected", i.dir) //nolint: goerr113
	}

	return nil
}

func (i *imgFileStorage) imagePath(imageName string) string {
	return filepath.Join(i.dir, imageNameToFileName(imageName))
}

func imageNameToFileName(str string) string {
	sb := strings.Builder{}
	sb.WriteString(imageSlug.ReplaceAllString(str, "_"))
	sb.WriteString("_")
	sum := sha256.New().Sum([]byte(str))
	sb.WriteString(fmt.Sprintf("%x", sum[:4]))
	sb.WriteString(".cache")

	return sb.String()
}
