# scrape-phabricator-macros
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Ftedkornish%2Fscrape-phabricator-macros.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Ftedkornish%2Fscrape-phabricator-macros?ref=badge_shield)


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


[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Ftedkornish%2Fscrape-phabricator-macros.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Ftedkornish%2Fscrape-phabricator-macros?ref=badge_large)