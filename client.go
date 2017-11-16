package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type (
	client struct{ host, key string }
	macro  struct{ uri, name string }
)

type macroImage string

func (c client) methodURL(apiMethod string) string {
	return fmt.Sprintf("%s/api/%s?%s",
		c.host,
		apiMethod,
		url.Values{"api.token": []string{c.key}}.Encode(),
	)
}

func (c client) getMacros() ([]macro, error) {
	var payload struct {
		Result map[string]struct {
			URI string `json:"uri"`
		} `json:"result"`
	}

	resp, err := http.Get(c.methodURL("macro.query"))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	var macros []macro
	for macroName, payload := range payload.Result {
		macros = append(macros, macro{uri: payload.URI, name: macroName})
	}

	return macros, nil
}

func (c client) getMacroImage(macroName string) (macroImage, error) {
	return "", nil
}
