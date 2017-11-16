package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	liburl "net/url"
)

type (
	client struct{ host, key string }
	macro  struct{ uri, name, filePHID string }

	macroImage struct {
		macro
		body []byte
	}
)

func (c client) methodURL(apiMethod string, params map[string]string) string {
	return c.urlWithToken(c.host+"/api/"+apiMethod, params)
}

func (c client) urlWithToken(url string, params map[string]string) string {
	values := liburl.Values{"api.token": []string{c.key}}
	for key, val := range params {
		values[key] = []string{val}
	}
	return fmt.Sprintf("%s?%s", url, values.Encode())
}

func (c client) getMacros() ([]macro, error) {
	var payload struct {
		Result map[string]struct {
			URI      string `json:"uri"`
			FilePHID string `json:"filePHID"`
		} `json:"result"`
	}

	resp, err := http.Get(c.methodURL("macro.query", nil))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	var macros []macro
	for macroName, payload := range payload.Result {
		macros = append(macros, macro{
			uri:      payload.URI,
			name:     macroName,
			filePHID: payload.FilePHID,
		})
	}

	return macros, nil
}

func (c client) getMacroImage(macro macro) (macroImage, error) {
	resp, err := http.Get(c.methodURL("file.download", map[string]string{
		"phid": macro.filePHID,
	}))
	if err != nil {
		return macroImage{}, err
	}
	defer resp.Body.Close()

	var payload struct {
		Result string `json:"result"` // a base64-encoded string
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return macroImage{}, err
	}

	body, err := base64.StdEncoding.DecodeString(payload.Result)
	if err != nil {
		return macroImage{}, err
	}

	return macroImage{macro: macro, body: body}, nil
}
