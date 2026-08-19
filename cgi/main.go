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

var (
	tRe   = regexp.MustCompile(`(?is:<title>(.*?)</title>)`)
	numRe = regexp.MustCompile(`^\d+$`)
	reFlg = regexp.MustCompile(`^r=`)
	dbFlg = regexp.MustCompile(`^d=`)
	hpFlg = regexp.MustCompile(`^h=`)
	icFlg = regexp.MustCompile(`^i=`)
	wsRe  = `(\s|&nbsp;|</?[^>]+>)+`
	dbg   = false
	igncs = false
)

func dbgPrint(w http.ResponseWriter, s string) {
	if dbg {
		fmt.Fprint(w, s)
	}
}

func title(w http.ResponseWriter, fn string, itm []string) string {
	f, err := os.Open(fsroot + fn)
	if err != nil {
		fmt.Fprintf(w, "error: %v<br>\n", err)
		return ""
	}
	defer f.Close()
	r := bufio.NewReader(f)
	_, err = f.Stat()
	if err != nil {
		fmt.Printf("stat error: %w\n", err)
		return ""
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		fmt.Fprintf(w, "error: %v<br>\n", err)
		return ""
	}
	t := tRe.FindStringSubmatch(string(buf))
	title := path.Base(fn)
	dbgPrint(w, fmt.Sprintf("debug file name: '%v'<br>\n", title))
	if len(t) > 1 {
		title = t[1]
		dbgPrint(w, fmt.Sprintf("debug title: '%v'<br>\n", title))
	}
	for _, i := range itm {
		j := strings.Split(i, "+")
		var pat *regexp.Regexp
		var err error
		var spat string
		if len(j) > 1 {
			spat = strings.Join(j, wsRe)
		} else {
			spat = j[0]
		}
		if igncs {
			spat = "(?i)" + spat
		}
		pat, err = regexp.Compile(spat)
		if err != nil {
			fmt.Fprintf(w, "re compile error: %v<br>\n", err)
			return ""
		}
		dbgPrint(w, fmt.Sprintf("debug pat: '%v'<br>\n", pat))
		t = pat.FindStringSubmatch(string(buf))
		if len(t) > 0 {
			dbgPrint(w, fmt.Sprintf("t: %v<br>\n", t[0]))
		} else {
			return ""
		}
	}
	return title
}

func prune(s string) bool {
	if len(s) < 4 {
		fmt.Printf("Ignore %s: too short<br>\n", s)
		return false
	}
	if numRe.MatchString(s) {
		fmt.Printf("Ignore %s: skipping whole numbers<br>\n", s)
		return false
	}
	return true
}

func main() {
	re, hlp := false, false
	if err := cgi.Serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Content-Type", "text/html; charset=utf-8")
		f, err := os.Open(infile)
		if err != nil {
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
		fmt.Fprintf(w, "<h1>%s search page</h1>\n", spagename)
		buf := make([]byte, 64)
		defer r.Body.Close()
		n, err := r.Body.Read(buf)
		if err != nil && err.Error() != "EOF" {
			fmt.Fprintf(w, "error: %v<br>\n", err)
		} else {
			s := string(buf[:n])
			arg := strings.Split(s, "&")
			s = arg[0]
			for _, k := range arg[1:] {
				if reFlg.MatchString(k) {
					re = true
				} else if dbFlg.MatchString(k) {
					dbg = true
				} else if hpFlg.MatchString(k) {
					hlp = true
				} else if icFlg.MatchString(k) {
					igncs = true
				}
			}
			dbgPrint(w, fmt.Sprintf("arg: %v<br>\n", arg))
			s, _ = strings.CutPrefix(s, "w=")
			s = strings.TrimSpace(s)
			s, err = url.QueryUnescape(s)
			if err != nil && err.Error() != "EOF" {
				fmt.Fprintf(w, "error: %v<br>\n", err)
			}
			S := s
			s = strings.ToLower(s)
			if s != "" {
				Items := strings.Split(S, " ")
				items := []string{}
				allitems := strings.Split(s, " ")
				for _, i := range allitems {
					for _, j := range strings.Split(i, "+") {
						if prune(j) {
							items = append(items, j)
						}
					}
				}
				nit := len(items)
				dbgPrint(w, fmt.Sprintf("items: %v, len: %v<br>\n", items, nit))
				dbgPrint(w, fmt.Sprintf("Items: %v, len: %v<br>\n", Items, len(Items)))
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
							f := l[1]
							if _, ok := skipre[f]; !ok {
								skipre[f] = make(map[string]bool)
							}
							if skipre[f][i] {
								break
							}
							skipre[f][i] = true
							hit[f] += 1
							if hit[f] == nit {
								t := title(w, l[1], Items)
								if t == "" {
									break
								}
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
		fmt.Fprintf(w, `<form method="post" action="/cgi-bin/%s">`, exename)
		fmt.Fprintf(w, `<input type="text" name="w" value="">`)
		fmt.Fprintf(w, `<input type="submit" value="Search">`)
		fmt.Fprintf(w, "<br>regexp mode: ")
		checked := map[bool]string{false: "", true: " CHECKED"}
		fmt.Fprintf(w, `<input type="checkbox"%s name="r"`, checked[re])
		fmt.Fprintf(w, "<br><br>ignore case: ")
		ignorecase := map[bool]string{false: "", true: " CHECKED"}
		fmt.Fprintf(w, `<input type="checkbox"%s name="i"`, ignorecase[igncs])
		fmt.Fprintf(w, "<br><br>show help: ")
		help := map[bool]string{false: "", true: " CHECKED"}
		fmt.Fprintf(w, `<input type="checkbox"%s name="h"`, help[hlp])
		fmt.Fprintf(w, "<br><br>show debug: ")
		debug := map[bool]string{false: "", true: " CHECKED"}
		fmt.Fprintf(w, `<input type="checkbox"%s name="d"`, debug[dbg])
		fmt.Fprintf(w, "<br></form>\n")
		if hlp {
			fmt.Fprintf(w, "<br>The search applies to a list of words, at least 4 chars long, excluding whole numbers.\n")
			fmt.Fprintf(w, "<br>Words not satifying these restrictions are ignored, which is reported..\n")
			fmt.Fprintf(w, "<br>Patterns are space separated, and 'AND'ed.\n")
			fmt.Fprintf(w, "<br>In the default mode, they must match exactly,\n")
			fmt.Fprintf(w, "<br>&nbsp;the matches are per page, and independent; the order doesn't matter.\n")
			fmt.Fprintf(w, "<br>However, patterns of  '+' -separated words are treated as sequences.\n")
			fmt.Fprintf(w, "<br>In the regexp mode, patterns are not anchored, so they may match the middle of words.\n")
			fmt.Fprintf(w, "<br>The '|' operator may be used to match alternatives, and '.' to match any single char. '\\b' means word boundary.\n")
			fmt.Fprintf(w, "<br>Examples:\n")
			fmt.Fprintf(w, "<br>'love|amour|любовь' matches any of the 3 words, but also 'amours', 'lover' or 'clove'.\n")
			fmt.Fprintf(w, "<br>Use '\b(love|amour)\b' to restrict to the exact words (bug with parentheses and cyrillic...)\n")
			fmt.Fprintf(w, "<br>'[eé]vidence' to accept both English and French spellings.\n")
			fmt.Fprintf(w, "<br>'différ.nce\b' to accept Derrida's variant.\n")
		}
		fmt.Fprintf(w, "</body>\n")
		fmt.Fprintf(w, "</html>\n")
	})); err != nil {
		fmt.Println(err)
	}
}
