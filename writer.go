package main

import (
	"io/ioutil"
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
