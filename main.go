package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/cheggaaa/pb"
)

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
		fmt.Println("Please specify a Phabricator host with the -host flag")
		os.Exit(1)
	} else if *key == "" {
		fmt.Println("Please specify an API key with the -key flag")
		os.Exit(1)
	} else if *dir == "" {
		fmt.Println("Please specify an output directory with the -dir flag")
		os.Exit(1)
	}

	return config{
		client:               client{host: *host, key: *key},
		writer:               writer{dir: *dir},
		numConcurrentFetches: *numConcurrentFetches,
	}, nil
}

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

func main() {
	config, err := getConfig()
	if err != nil {
		fmt.Println("Failed to get config:", err)
		os.Exit(1)
	}

	// Test the writer before sending any HTTP requests so we can short-circuit.
	if err := config.writer.test(); err != nil {
		fmt.Println("Can't write to specified directory:", err)
		os.Exit(1)
	}

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

	go handleImage(channels, errorSet, wg, config.writer, bar)

	for _, macro := range macros {
		channels.pending <- macro
		wg.Add(1)
	}

	wg.Wait()

	// When we're done, close all the channels and print all the errors.
	channels.closeAll()
	errorSet.printAll()
}

func getMacroImage(
	client client,
	channels *channels,
) {
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
			bar.Increment()
			wg.Done()
		case image := <-channels.images:
			if err := writer.writeImage(image); err != nil {
				errorSet.add(err)
			}
			bar.Increment()
			wg.Done()
		}
	}
}
