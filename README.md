search/crawl/findlinks.go produces a key/value text file with words and paths relative to a local root.
The local root, as well as words and paths to be skipped are configured in search/crawl/findlinks.json.
findlinks.json contains example values to be overwritten.

The output text file is consumed by the cgi script installed from search/cgi/main.go.

Setup in search/cgi/main.go:

- two constants: `pfx` and `infile`
  `pfx` is the url mapped to the root defined in findlinks.json

Installation (as an example, remotely built for a raspberry pi named `berry`):

~> cd ~/git/search/cgi
cgi> GOOS=linux GOARCH=arm64 GOARM=7 go build .
cgi> scp cgi berry:/tmp/

On berry:
~> sudo mv /tmp/cgi /usr/lib/cgi-bin/search

Instructions:

~> cd ~/git/search/crawl/
crawl> go run findlinks.go > /tmp/fl.out
crawl> scp /tmp/fl.out berry:/tmp/

Then on berry:
~> sudo mv /tmp/fl.out /usr/lib/cgi-bin/
