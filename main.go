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

	bar := pb.New(len(macros))
	bar.Start()

	wg := new(sync.WaitGroup)
	pendingMacros := make(chan macro)
	errChan := make(chan error)
	imageChan := make(chan macroImage)
	var errors []error

	for i := 0; i < config.numConcurrentFetches; i++ {
		go func() {
			for {
				macro := <-pendingMacros
				imageFile, err := config.client.getMacroImage(macro)
				if err != nil {
					errChan <- err
				} else {
					imageChan <- imageFile
				}
			}
		}()
	}

	go func() {
		for {
			select {
			case err := <-errChan:
				errors = append(errors, err)
				bar.Increment()
			case image := <-imageChan:
				if err := config.writer.writeImage(image); err != nil {
					errors = append(errors, err)
				}
				bar.Increment()
			}
		}
	}()

	for _, macro := range macros {
		pendingMacros <- macro
		wg.Add(1)
	}

	wg.Wait()
}
