package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	liburl "net/url"
)

type (
	client struct{ host, key string }
	macro  struct{ uri, name string }

	macroImage struct {
		macro
		body []byte
	}
)

func (c client) methodURL(apiMethod string) string {
	return c.urlWithToken(c.host + "/api/" + apiMethod)
}

func (c client) urlWithToken(url string) string {
	values := liburl.Values{"api.token": []string{c.key}}
	return fmt.Sprintf("%s?%s", url, values.Encode())
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

func (c client) getMacroImage(macro macro) (macroImage, error) {
	resp, err := http.Get(c.urlWithToken(macro.uri))
	if err != nil {
		return macroImage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return macroImage{}, err
	}

	return macroImage{macro: macro, body: body}, nil
}
