package main

import (
	"os"
	"path/filepath"
)

var (
	pfx, infile, fsroot, exename, spagename string
)

func init() {
	exename = filepath.Base(os.Args[0])
	spagename = "berry"
	pfx = `https://berry314.girod.fi/rnotes/`
	infile = `search.out`
	fsroot = "/home/marc/public_html/"
	if exename == "svenska" {
		spagename = "svenska"
		pfx = `https://berry314.girod.fi/rnotes/l/`
		infile = `svenska.out`
		fsroot = "/home/marc/public_html/l/"
	}
}
