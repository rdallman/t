package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
)

// TODO literals vs regex
// TODO binaries maybe ignore early on

func usage() {
	fmt.Println(`usage: t pattern [file]`)
}

func main() {
	flag.Parse()
	narg := flag.NArg()
	args := flag.Args()
	if narg < 1 {
		usage()
		os.Exit(1)
	}

	pat := args[0]

	// get file infos to make sure we only have real files
	files, errc := gatherFiles(args[1:])
	go func() {
		err := <-errc
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	pres := preBmBc(pat)
	for f := range files {
		// TODO concurrent pipeline
		txt, err := ioutil.ReadFile(f)
		if err != nil {
			continue // TODO err chan ?
		}
		// TODO concurrent pipeline
		found := search(pat, string(txt), pres)
		pnt(f, string(txt), found, 0)
	}
}

// TODO custom ReadDir that uses .git, .hg knowledge and only spits out file files
func gatherFiles(args []string) (<-chan string, <-chan error) {
	paths := make(chan string)
	errc := make(chan error, 1)

	go func() {
		defer close(paths)

		fi, _ := os.Stdin.Stat()
		if fi.Mode()&os.ModeNamedPipe != 0 || fi.Size() > 0 {
			paths <- os.Stdin.Name()
			return // don't do anything else
		}

		if len(args) == 0 {
			args = append(args, ".")
		}

		// TODO find rooted .hg, .git or .bzr, use that knowledge
		for _, fname := range args {
			errc <- filepath.Walk(fname, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}

				paths <- path
				return nil
			})
		}

	}()
	return paths, errc
}

const (
	ASIZE = 256
)

// TODO don't just use \n, give char max context, too for binaries and long lines
func pnt(fname, txt string, found map[int]struct{}, ctxLines int) {
	var ln []rune // TODO use buffer and clear?
	var nameprinted, match bool
	var lnum int
	for i, char := range txt {
		if _, ok := found[i]; ok {
			match = true
		}
		ln = append(ln, char)
		if char == '\n' { // put each new line at end of ctx
			lnum++ // 1 indexed
			if match == true {
				if !nameprinted {
					fmt.Println(fname)
					nameprinted = true
				}
				fmt.Printf("%d: %s", lnum, string(ln))
			}
			ln = nil
			match = false
		}
	}
}

func preBmBc(pat string) []int {
	bmBc := make([]int, ASIZE)
	for i := 0; i < ASIZE; i++ {
		bmBc[i] = -1
	}
	for i := 0; i < len(pat); i++ {
		bmBc[pat[i]] = i
	}
	return bmBc
}

func search(pat, txt string, preBmBc []int) (found map[int]struct{}) {
	m := len(pat)
	n := len(txt)
	var skip int
	found = make(map[int]struct{})
	for i := 0; i <= n-m; i += skip {
		skip = 0
		for j := m - 1; j >= 0; j-- {
			if pat[j] != txt[i+j] {
				skip = int(math.Max(1, float64(j-preBmBc[txt[i+j]])))
				break
			}
		}
		if skip == 0 {
			found[i] = struct{}{}
			skip++
		}
	}
	return found
}
