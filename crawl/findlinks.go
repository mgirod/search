// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// See page 243.

// Crawl3 crawls web links starting with the command-line arguments.
//
// This version uses bounded parallelism.
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/scanner"
	"time"
	"unicode/utf8"

	nurl "net/url"

	cfg "github.com/gookit/config/v2"
	"golang.org/x/net/html"
)

var root string
var skipword []string
var skippath []string

func init() {
	err := cfg.LoadFiles("findlinks.json")
	if err != nil {
		panic(err)
	}
	root = cfg.String("root")
	skipword = cfg.Strings("skipword")
	skippath = cfg.Strings("skippath")
}

var rroot = regexp.MustCompile("^" + root)
var ranchor = regexp.MustCompile("[#?].*$")
var skip = regexp.MustCompile(`^(https?://|(mailto|ftp|news):|/(vob/|cgi-bin/(cchist|man)))|\.ps$`)
var slash = regexp.MustCompile("^/")
var skipscan = regexp.MustCompile(`^(invalid digit '\d' in octal literal|(exponent|hexadecimal literal) has no digits)|must separate successive digits`)

// Extract makes an HTTP GET request to the specified URL, parses
// the response as HTML, and returns the links in the HTML document.
func Extract(url string) ([]string, error) {
	page := strings.TrimPrefix(url, "file://")
	if skip.MatchString(page) {
		return nil, nil
	}
	f, err := os.Open(page)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dir := filepath.Dir(page)

	doc, err := html.Parse(bufio.NewReader(f))
	if err != nil {
		return nil, fmt.Errorf("parsing %s as HTML: %v", url, err)
	}

	var links []string
	visitNode := func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key != "href" {
					continue
				}
				link, err := nurl.Parse(a.Val)
				if err != nil {
					continue // ignore bad URLs
				}
				fn := link.String()
				if skip.MatchString(fn) {
					continue
				}
				if !slash.MatchString(fn) {
					fn = filepath.Clean(dir + "/" + fn)
				}
				fn = ranchor.ReplaceAllString(fn, "")
				info, err := os.Stat(fn)
				if err != nil {
					log.Print(err)
					continue
				}
				if info.IsDir() {
					fn = strings.TrimRight(fn, "/") + "/index.html"
					if _, err = os.Stat(fn); err != nil {
						continue
					}
				}
				links = append(links, fn)
			}
		}
	}
	seen := make(map[string]bool)
	for _, i := range skipword {
		seen[i] = true
	}
	fn := strings.TrimLeft(rroot.ReplaceAllString(page, ""), "/")
	forEachNode(doc, &seen, fn, visitNode, exttok)
	return links, nil
}

// Copied from gopl.io/ch5/outline2.
func forEachNode(n *html.Node, s *map[string]bool, fn string, pre func(n *html.Node), post func(n *html.Node, s *map[string]bool, fn string)) {
	if pre != nil {
		pre(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && n.Data == `head` {
			continue
		}
		forEachNode(c, s, fn, pre, post)
	}
	if post != nil {
		post(n, s, fn)
	}
}

func crawl(url string) []string {
	m := rroot.MatchString(url)
	if !m {
		return nil
	}
	// fmt.Println(url)
	list, err := Extract(url)
	if err != nil {
		log.Print(err)
	}
	return list
}

func exttok(n *html.Node, seen *map[string]bool, fn string) {
	if n.Type == html.TextNode {
		var s scanner.Scanner
		s.Init(strings.NewReader(n.Data))
		s.Whitespace |= 1 << '\''
		s.Whitespace |= 1 << '"'
		s.Whitespace |= 1 << '/'
		s.Mode ^= scanner.ScanFloats
		s.Error = func(s *scanner.Scanner, m string) {
			if skipscan.MatchString(m) {
				return
			}
			fmt.Fprintf(os.Stderr, "%s: %s\n", fn, m)
		}
		for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
			t := strings.ToLower(s.TokenText())
			if !utf8.ValidString(t) || utf8.RuneCount([]byte(t)) <= 3 || (*seen)[t] {
				continue
			}
			(*seen)[t] = true
			fmt.Println(t, fn)
		}
	}
}

// !+
func main() {
	worklist := make(chan []string)  // lists of URLs, may have duplicates
	unseenLinks := make(chan string) // de-duplicated URLs
	var n sync.WaitGroup

	// Add command-line arguments to worklist.
	n.Add(1)
	go func() {
		worklist <- []string{root + "/index.html"}
	}()

	go func() {
		for link := range unseenLinks {
			defer n.Done()
			go func() {
				// fmt.Println("from unseenLinks: ", link)
				n.Add(1)
				list := crawl(link)
				worklist <- list
			}()
		}
	}()
	go func() {
		n.Wait()
		close(worklist)
	}()

	// The main goroutine de-duplicates worklist items
	// and sends the unseen ones to the crawlers.
	seen := make(map[string]bool)
	for _, i := range skippath {
		seen[i] = true
	}
	for list := range worklist {
		for _, link := range list {
			if !seen[link] {
				seen[link] = true
				unseenLinks <- link
			}
		}
		n.Done()
	}
	fmt.Fprintln(os.Stderr, time.Now())
}

//!-
