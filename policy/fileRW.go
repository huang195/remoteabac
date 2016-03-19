package policy

import (
	"io/ioutil"
	"log"
)

type FileRW struct {
	Path string
}

var _ ReaderWriter = (*FileRW)(nil)

func NewFileRW(path string) (*FileRW, error) {
	log.Printf("Loading policy file from a local file: %s\n", path)
	return &FileRW{
		Path: path,
	}, nil
}

func (f *FileRW) Read() (string, error) {
	b, err := ioutil.ReadFile(f.Path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (f *FileRW) Write(content string) error {
	return ioutil.WriteFile(f.Path, []byte(content), 0644)
}
