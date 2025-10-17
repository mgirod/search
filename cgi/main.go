package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/cgi"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
)

const (
	pfx    = `https://mydomain/myroot/` // expose `root` in findlinks.json
	infile = `fl.out`
)

var (
	tRe = regexp.MustCompile(`(?is:<title>(.*?)</title>)`)
)

func title(w http.ResponseWriter, fn string) string {
	f, err := os.Open("/home/marc/public_html/" + fn)
	if err != nil {
		fmt.Fprintf(w, "error: %v<br>\n", err)
		return ""
	}
	defer f.Close()
	r := bufio.NewReader(f)
	sz := 512
	if fi, er := f.Stat(); er == nil && fi.Size() < 512 {
		sz = int(fi.Size())
	}
	buf := make([]byte, sz)
	n, err := io.ReadFull(r, buf)
	if err != nil || n == 0 {
		fmt.Fprintf(w, "n: %d, error: %v<br>\n", n, err)
		return ""
	}
	// fmt.Fprintf(w, "debug n: %d, buf: %v<br>\n", n, string(buf[:n]))
	t := tRe.FindStringSubmatch(string(buf[:n]))
	if len(t) < 2 {
		// fmt.Fprintf(w, "t: %v<br>\n", t)
		return path.Base(fn)
	}
	return t[1]
}

func main() {
	if err := cgi.Serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Content-Type", "text/html; charset=utf-8")
		f, err := os.Open(infile)
		if err != nil {
			// fmt.Fprintf(w, "<title>Error</title>\n")
			fmt.Fprintf(w, "%s\n", err.Error())
			panic(err.Error())
		}
		db := bufio.NewScanner(f)
		defer f.Close()
		fmt.Fprintf(w, "<html>\n")
		fmt.Fprintf(w, "<head>\n")
		fmt.Fprintf(w, "<title>Search</title>\n")
		fmt.Fprintf(w, "</head>\n")
		fmt.Fprintf(w, "<body>\n")
		fmt.Fprintf(w, "<h1>berry search page</h1>\n")
		buf := make([]byte, 64)
		defer r.Body.Close()
		n, err := r.Body.Read(buf)
		re := false
		if err != nil && err.Error() != "EOF" {
			fmt.Fprintf(w, "error: %v<br>\n", err)
		} else {
			s := string(buf[:n])
			arg := strings.Split(s, "&")
			s = arg[0]
			re = len(arg) > 1
			s, _ = strings.CutPrefix(s, "w=")
			s, err = url.QueryUnescape(s)
			if err != nil && err.Error() != "EOF" {
				fmt.Fprintf(w, "error: %v<br>\n", err)
			}
			s = strings.ToLower(s)
			s = strings.TrimSpace(s)
			if s != "" {
				items := strings.Split(s, " ")
				nit := len(items)
				hit := make(map[string]int)
				//successfully match re only once per file
				skipre := make(map[string]map[string]bool)
				remap := make(map[string]*regexp.Regexp, nit)
				if re {
					for _, i := range items {
						remap[i] = regexp.MustCompile(i)
					}
				}
				for db.Scan() {
					// fmt.Fprintf(w, "debug: %v<br>\n", db.Text())
					l := strings.SplitN(db.Text(), " ", 2)
					// fmt.Fprintf(w, "debug: %v<br>\n", l[0])
					for _, i := range items {
						match := false
						if re {
							match = remap[i].MatchString(l[0])
						} else {
							match = l[0] == i
						}
						if match {
							t := title(w, l[1])
							if t == "" {
								t = l[0]
							}
							if _, ok := skipre[t]; !ok {
								skipre[t] = make(map[string]bool)
							}
							if skipre[t][i] {
								break
							}
							skipre[t][i] = true
							hit[t] += 1
							if hit[t] == nit {
								fmt.Fprintf(w, "%s<br>\n", `<a href="`+pfx+l[1]+`">`+t+"</a>")
							}
						}
					}
				}
				if err := db.Err(); err != nil {
					fmt.Fprintf(w, "scan error: %v<br>\n", err)
				}
			}
		}
		fmt.Fprintf(w, "<br><hr>")
		fmt.Fprintf(w, `<form method="post" action="/cgi-bin/search">`)
		fmt.Fprintf(w, `<input type="text" name="w" value="">`)
		fmt.Fprintf(w, `<input type="submit" value="Search">`)
		fmt.Fprintf(w, "<br> regexp mode: ")
		checked := map[bool]string{false: "", true: " CHECKED"}
		fmt.Fprintf(w, `<input type="checkbox"%s name="r"`, checked[re])
		fmt.Fprintf(w, "<br></form>\n")
		fmt.Fprintf(w, "</body>\n")
		fmt.Fprintf(w, "</html>\n")
	})); err != nil {
		fmt.Println(err)
	}
}
