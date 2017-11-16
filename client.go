package main

type client struct{ host, key string }

type macro struct {
	name string
}

type macroImage string

func (c client) getMacros() ([]macro, error) {
	return nil, nil
}

func (c client) getMacroImage(macroName string) (macroImage, error) {
	return "", nil
}
