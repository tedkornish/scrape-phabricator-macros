# scrape-phabricator-macros

This project produces a binary which scrapes macros from a specified Phabricator instance and outputs them as gifs to a specified directory in a local filesystem.

## Installation

```
go get -u github.com/tedkornish/scrape-phabricator-macros
```

## Execution

To run (assuming `$GOPATH/bin` is in your `$PATH`):

```
mkdir /tmp/macros # the output directory must exist beforehand
scrape-phabricator-macros -host="https://code.cleargraph.io" -key="cli-my-key-here" -dir="/tmp/macros" -numConcurrentFetches=50
```

The `-host`, `-key`, and `-dir` flags are required; the `-numConcurrentFetches` flag is optional and defaults to 50.

## License

MIT.
