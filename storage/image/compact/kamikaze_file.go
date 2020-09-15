package compact

import (
	"io/ioutil"
	"os"
)

// kamikazeFile removes itself on Closing.
type kamikazeFile struct {
	file *os.File
}

func (t *kamikazeFile) Seek(offset int64, whence int) (int64, error) {
	return t.file.Seek(offset, whence)
}

func newKamikazeFile() (*kamikazeFile, error) {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}

	return &kamikazeFile{file: file}, nil
}

func (t *kamikazeFile) Write(p []byte) (n int, err error) {
	return t.file.Write(p)
}

func (t *kamikazeFile) Read(p []byte) (n int, err error) {
	return t.file.Read(p)
}

func (t *kamikazeFile) Close() error {
	closeErr := t.file.Close()
	removeErr := os.Remove(t.file.Name())
	if closeErr != nil {
		return closeErr
	}
	if removeErr != nil {
		return closeErr
	}

	return nil
}
