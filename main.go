package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	liburl "net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/cheggaaa/pb"
)

func main() {

	// Get config from flags. This struct contains client and writer abstractions
	// which give us access to the outside world - specifically, to the
	// Phabricator HTTP API and to the local filesystem.
	config, err := getConfig()
	if err != nil {
		fmt.Println("Failed to get config:", err)
		os.Exit(1)
	}

	// Test the writer before sending any HTTP requests so we can short-circuit;
	// we want to avoid sending a single HTTP request if we can anticipate a local
	// filesystem error, such as incorrect permissions.
	if err := config.writer.test(); err != nil {
		fmt.Println("Can't write to specified directory:", err)
		os.Exit(1)
	}

	// Get a list of all macros so we know which images to fetch.
	macros, err := config.client.getMacros()
	if err != nil {
		fmt.Println("Failed to fetch macros:", err)
		os.Exit(1)
	}

	var (
		bar      = pb.New(len(macros))
		wg       = new(sync.WaitGroup)
		errorSet = makeErrorSet()
		channels = makeChannels()
	)

	bar.Start()

	// Start as many goroutines fetching images as specified in the config.
	for i := 0; i < config.numConcurrentFetches; i++ {
		go getMacroImage(config.client, channels)
	}

	// Wait for image bytes to come through so we can write them to files locally.
	go handleImage(channels, errorSet, wg, config.writer, bar)

	// Actually queue the macros up for retrieval.
	for _, macro := range macros {
		channels.pending <- macro
		wg.Add(1)
	}

	wg.Wait()

	// When we're done, close all the channels and print all the errors.
	channels.closeAll()
	errorSet.printAll()
}

type config struct {
	client               client
	writer               writer
	numConcurrentFetches int
}

func getConfig() (config, error) {
	host := flag.String("host", "", "the host of the Phabricator instance")
	key := flag.String("key", "", "the Conduit API key for Phabricator")
	dir := flag.String("dir", "", "the output directory for the macro images")
	numConcurrentFetches := flag.Int(
		"numConcurrentFetches",
		10,
		"number of HTTP requests to have in-flight concurrently",
	)

	flag.Parse()

	if *host == "" {
		return config{}, errors.New("please specify a Phabricator host with the -host flag")
	} else if *key == "" {
		return config{}, errors.New("please specify an API key with the -key flag")
	} else if *dir == "" {
		return config{}, errors.New("please specify an output directory with the -dir flag")
	}

	return config{
		client:               client{host: *host, key: *key},
		writer:               writer{dir: *dir},
		numConcurrentFetches: *numConcurrentFetches,
	}, nil
}

// A collection of channels for queueing the retrieval of macros and writing of
// images.
type channels struct {
	pending chan macro
	errors  chan error
	images  chan macroImage
}

func (c *channels) closeAll() {
	close(c.pending)
	close(c.errors)
	close(c.images)
}

func makeChannels() *channels {
	return &channels{
		pending: make(chan macro),
		errors:  make(chan error),
		images:  make(chan macroImage),
	}
}

// A concurrency-safe list of errors.
type errorSet struct {
	mu     *sync.Mutex
	errors []error
}

func makeErrorSet() *errorSet {
	return &errorSet{
		mu:     new(sync.Mutex),
		errors: make([]error, 0),
	}
}

func (set *errorSet) add(err error) {
	set.mu.Lock()
	set.errors = append(set.errors)
	set.mu.Unlock()
}

// Print each error on a new line to stdout.
func (set *errorSet) printAll() {
	set.mu.Lock()
	if len(set.errors) > 0 {
		fmt.Printf("%d errors:\n", len(set.errors))
		for _, error := range set.errors {
			fmt.Println("-", error)
		}
	}
	set.mu.Unlock()
}

// writer abstracts away interaction with the local filesystem.
type writer struct {
	dir string
}

// Write an image in .gif format to the filesystem.
func (w writer) writeImage(image macroImage) error {
	err := ioutil.WriteFile(filepath.Join(w.dir, image.name+".gif"), image.body, 0600)
	if err != nil {
		return err
	}
	return nil
}

// Write a test file to the filesystem at the specified directory just to see if
// we can.
func (w writer) test() error {
	testFilePath := filepath.Join(w.dir, "test")
	if err := ioutil.WriteFile(testFilePath, []byte("test"), 0600); err != nil {
		return err
	}
	return os.Remove(testFilePath)
}

// macro encodes for the name and file PHID (Phabricator ID) of a macro.
type macro struct{ name, filePHID string }

// macroImage is a macro with its contents encoded as raw bytes.
type macroImage struct {
	macro
	body []byte
}

type client struct {
	host, key string
}

// Return the url for a GET request to the specified Phabricator API method.
func (c client) methodURL(apiMethod string, params map[string]string) string {
	return c.urlWithToken(c.host+"/api/"+apiMethod, params)
}

// Given a raw URL string and arbitrary query params, add the client's API token
// to the params in the proper key and return the full encoded URL.
func (c client) urlWithToken(url string, params map[string]string) string {
	values := liburl.Values{"api.token": []string{c.key}}
	for key, val := range params {
		values[key] = []string{val}
	}
	return fmt.Sprintf("%s?%s", url, values.Encode())
}

// Retrieve a list of macros from the client's Phabricator instance
// using the client's API key.
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
		macros = append(macros, macro{name: macroName, filePHID: payload.FilePHID})
	}

	return macros, nil
}

// Retrieve a given macro's image.
func (c client) getMacroImage(macro macro) (macroImage, error) {
	resp, err := http.Get(c.methodURL("file.download", map[string]string{
		"phid": macro.filePHID,
	}))
	if err != nil {
		return macroImage{}, err
	}
	defer resp.Body.Close()

	// Oddly, images from the file.download endpoint come as base64-encoded
	// strings, so we'll need to decode those before writing the bytes to disk.
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

// Loop forever, reading macros off the pending channel, sending the request
// to get the corresponding image, and either passing the image or an error
// back via a channel.
func getMacroImage(client client, channels *channels) {
	for {
		macro := <-channels.pending
		imageFile, err := client.getMacroImage(macro)
		if err != nil {
			channels.errors <- err
		} else {
			channels.images <- imageFile
		}
	}
}

// Loop forever, reading either images or errors. If an image, write to the
// proper local file. Increment the bar and decrement the WaitGroup regardless.
func handleImage(
	channels *channels,
	errorSet *errorSet,
	wg *sync.WaitGroup,
	writer writer,
	bar *pb.ProgressBar,
) {
	for {
		select {
		case err := <-channels.errors:
			errorSet.add(err)
		case image := <-channels.images:
			if err := writer.writeImage(image); err != nil {
				errorSet.add(err)
			}
		}
		bar.Increment()
		wg.Done()
	}
}
