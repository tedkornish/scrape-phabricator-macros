package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

type writer struct {
	dir string
}

func (w writer) writeImage(image macroImage) error {
	err := ioutil.WriteFile(filepath.Join(w.dir, image.name+".gif"), image.body, 0600)
	if err != nil {
		return err
	}
	return nil
}

func (w writer) test() error {
	testFilePath := filepath.Join(w.dir, "test")
	if err := ioutil.WriteFile(testFilePath, []byte("test"), 0600); err != nil {
		return err
	}
	return os.Remove(testFilePath)
}
