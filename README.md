search/crawl/findlinks.go produces a key/value text file with words and paths relative to a local root.  
The local root, as well as words and paths to be skipped are configured in search/crawl/findlinks.json.  
This file may be (and in the example configuration, is) a symlink, allowing to switch between settings.  
One may thus produce distinct databases to support searching disjoint sub-domains.  
Note that if the domains are not strictly disjoint, the crawler will produce one and the same database.  
findlinks.json contains example values to be overwritten.

The output text file(s) is|are consumed by the cgi script installed from search/cgi/main.go.

Setup in search/cgi/main.go:

- `exename` is dynamically set to the name of the binary;
  this allows to alias it in order to support disjoint domains
- sets of constants:  
  `pfx` is the url mapped to the root defined in findlinks.json for web pages  
  `fsroot` is the file system directory under which to find the corresponding html files  
  `infile` is the name of the text database, local to the cgi 'script'
  `spagename` is a string used in the search page title

Installation (as an example, remotely built for a raspberry pi named `berry`):

    ~> cd ~/git/search/cgi
    cgi> GOOS=linux GOARCH=arm64 GOARM=7 go build -buildvcs=false .
    cgi> scp search berry:/tmp/

On berry (second command optional, to alias the binary):

    ~> sudo mv /tmp/search /usr/lib/cgi-bin/
    ~> sudo ln -f /usr/lib/cgi-bin/search /usr/lib/cgi-bin/svenska

Instructions:

    ~> cd ~/git/search/crawl/
    ~> crawl> readlink findlinks.json 
    search.json
    crawl> go run findlinks.go > /tmp/search.out
    crawl> scp /tmp/search.out berry:/tmp/

and optionally (here restoring search.json as the default setup):

    crawl> ln -sf svenska.json findlinks.json
    crawl> go run findlinks.go > /tmp/svenska.out
    crawl> scp /tmp/svenska.out berry:/tmp/
    crawl> ln -sf search.json findlinks.json

Then on berry:

    ~> sudo mv /tmp/search.out /usr/lib/cgi-bin/
    ~> sudo mv /tmp/svenska.out /usr/lib/cgi-bin/
