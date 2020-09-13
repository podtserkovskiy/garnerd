package compact

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/docker/docker/pkg/ioutils"
	log "github.com/sirupsen/logrus"
)

type manifestJSON []struct {
	Layers []string
}

type fileData struct {
	srcPath, tarPath string
}

type ImgStorage struct {
	dir string
	mu  sync.Mutex
}

func NewImgStorage(dir string) *ImgStorage {
	return &ImgStorage{dir: dir}
}

func (i *ImgStorage) Save(imageName string, imageDump io.Reader) error { // nolint: funlen,gocognit
	i.mu.Lock()
	defer i.mu.Unlock()
	defer i.cleanUp()

	imgMetaDir := filepath.Join(i.dir, "meta", imageNameToDirName(imageName))
	err := os.MkdirAll(imgMetaDir, os.ModePerm)
	if err != nil {
		return err
	}

	archive := tar.NewReader(imageDump)
	lastDir := "--initial-value--"
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// suppose any dir are layer-dir
		if header.FileInfo().IsDir() {
			lastDir = header.Name
			dstDir := filepath.Join(i.dir, "layers", lastDir)
			err = os.MkdirAll(dstDir, header.FileInfo().Mode())
			if err != nil {
				return err
			}

			continue
		}

		// check if it is layer's file
		if strings.HasPrefix(header.Name, lastDir) {
			dstFile := filepath.Join(i.dir, "layers", header.Name)
			dstFileStat, err := os.Stat(dstFile)
			switch {
			case os.IsNotExist(err):
				// create file
			case err != nil:
				return err
			case dstFileStat.Size() == header.FileInfo().Size():
				// file with the same name and size already exists
				continue
			}

			file, err := ioutils.NewAtomicFileWriter(dstFile, header.FileInfo().Mode())
			if err != nil {
				return err
			}

			_, cpErr := io.Copy(file, archive) // nolint: gosec
			if closeErr := file.Close(); closeErr != nil {
				return err
			}
			if cpErr != nil {
				return cpErr
			}

			continue
		}

		// everything else is metadata
		// recreate metadata
		file, err := ioutils.NewAtomicFileWriter(filepath.Join(imgMetaDir, header.Name), header.FileInfo().Mode())
		if err != nil {
			return err
		}

		if _, err = io.Copy(file, archive); err != nil { // nolint: gosec
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (i *ImgStorage) Load(imageName string) (io.ReadCloser, error) { // nolint: funlen
	i.mu.Lock()
	defer i.mu.Unlock()
	imgMetaDir := filepath.Join(i.dir, "meta", imageNameToDirName(imageName))
	if _, err := os.Stat(imgMetaDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("image '%v', does not exist", imageName) // nolint: goerr113
	}

	files, err := ioutil.ReadDir(imgMetaDir)
	if err != nil {
		return nil, err
	}

	toCopy := []fileData{}
	for _, file := range files {
		toCopy = append(toCopy, fileData{
			srcPath: filepath.Join(imgMetaDir, file.Name()),
			tarPath: file.Name(),
		})
	}

	manifestFile, err := os.Open(filepath.Join(imgMetaDir, "manifest.json"))
	if err != nil {
		return nil, err
	}

	var manifest manifestJSON
	if err = json.NewDecoder(manifestFile).Decode(&manifest); err != nil {
		return nil, err
	}

	for _, imageEntry := range manifest {
		for _, layerFile := range imageEntry.Layers {
			layerDirName := filepath.Dir(layerFile)
			layerDirPath := filepath.Join(i.dir, "layers", layerDirName)
			files, err := ioutil.ReadDir(layerDirPath)
			if err != nil {
				return nil, err
			}

			toCopy = append(toCopy, fileData{
				srcPath: layerDirPath,
				tarPath: filepath.Base(layerDirName) + string(filepath.Separator),
			})

			for _, file := range files {
				toCopy = append(toCopy, fileData{
					srcPath: filepath.Join(layerDirPath, file.Name()),
					tarPath: filepath.Join(layerDirName, file.Name()),
				})
			}
		}
	}

	outFile, err := newKamikazeFile()
	if err != nil {
		return nil, err
	}

	err = tarFiles(outFile, toCopy)
	if err != nil {
		return nil, err
	}

	if _, err = outFile.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	return outFile, nil
}

func (i *ImgStorage) Remove(imageName string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	defer i.cleanUp()

	imgMetaDir := filepath.Join(i.dir, "meta", imageNameToDirName(imageName))
	err := os.RemoveAll(imgMetaDir)
	if err != nil {
		return err
	}

	return nil
}

func (i *ImgStorage) IsExist(imageName string) (bool, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	imgMetaDir := filepath.Join(i.dir, "meta", imageNameToDirName(imageName))
	if _, err := os.Stat(imgMetaDir); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func (i *ImgStorage) RemoveNotIn(imageNames []string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	defer i.cleanUp()

	allowedSet := map[string]bool{}
	for _, name := range imageNames {
		allowedSet[imageNameToDirName(name)] = true
	}

	return filepath.Walk(filepath.Join(i.dir, "meta"), func(path string, info os.FileInfo, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}

		if filepath.Base(path) == "meta" {
			return nil
		}

		if !allowedSet[filepath.Base(path)] {
			return os.RemoveAll(path)
		}

		return filepath.SkipDir
	})
}

func (i *ImgStorage) Ping() error {
	stat, err := os.Stat(i.dir)
	if err != nil {
		return fmt.Errorf("ping '%s', %w", i.dir, err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("path '%s' is a file, directory is expected", i.dir) //nolint: goerr113
	}

	return nil
}

func (i *ImgStorage) cleanUp() {
	allowedLayers := map[string]bool{}
	err := filepath.Walk(filepath.Join(i.dir, "meta"), func(path string, info os.FileInfo, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Name() != "manifest.json" {
			return nil
		}

		manifestFile, err := os.Open(path)
		if err != nil {
			return err
		}

		var manifest manifestJSON
		if err = json.NewDecoder(manifestFile).Decode(&manifest); err != nil {
			return err
		}

		for _, imageEntry := range manifest {
			for _, layerFile := range imageEntry.Layers {
				allowedLayers[filepath.Dir(layerFile)] = true
			}
		}

		return nil
	})
	if err != nil {
		log.Warn("images cleanUp", err)
	}

	err = filepath.Walk(filepath.Join(i.dir, "layers"), func(path string, info os.FileInfo, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}

		if filepath.Base(path) == "layers" {
			return nil
		}

		if !allowedLayers[filepath.Base(path)] {
			return os.RemoveAll(path)
		}

		return filepath.SkipDir
	})
	if err != nil {
		log.Warn("images cleanUp", err)
	}
}

func imageNameToDirName(str string) string {
	return regexp.MustCompile(`\W+`).ReplaceAllString(str, "_")
}

func tarFiles(tmpTar io.Writer, toCopy []fileData) error {
	tw := tar.NewWriter(tmpTar)
	defer tw.Close()

	for _, data := range toCopy {
		fi, err := os.Stat(data.srcPath)
		if err != nil {
			return err
		}

		hdr, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		hdr.Name = data.tarPath
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if fi.Mode().IsDir() {
			continue
		}

		// add file to tar
		srcFile, err := os.Open(data.srcPath)
		if err != nil {
			return err
		}

		if _, err = io.Copy(tw, srcFile); err != nil {
			_ = srcFile.Close()

			return err
		}
		if err = srcFile.Close(); err != nil {
			return err
		}
	}

	return nil
}
