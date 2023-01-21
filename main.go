package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var log = betterlog{}

const helpString = `
Yet another file search tool. An equivalent of "find ... | egrep ..."
`

type stringArgs []string

func (a *stringArgs) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func (a *stringArgs) String() string {
	return fmt.Sprint(*a)
}

func main() {
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Usage = func() {
		fmt.Println(strings.Trim(helpString, " \n"))
		fmt.Println()
		flag.PrintDefaults()
	}
	var flagDebug bool
	flag.BoolVar(&flagDebug, "v", false, "verbose debug mode")
	var fileFilters stringArgs
	flag.Var(&fileFilters, "f", "file filters")
	var pathFilters stringArgs
	flag.Var(&pathFilters, "p", "path filters")
	var fileNegFilters stringArgs
	flag.Var(&fileNegFilters, "nf", "file negative filters")
	var pathNegFilters stringArgs
	flag.Var(&pathNegFilters, "np", "path negative filters")
	flag.Parse()

	log.Debug = flagDebug

	s, err := newSearchFromArgs(flag.Args(), fileFilters, fileNegFilters, pathFilters, pathNegFilters)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}
	log.Debugf("search: %s", s)
	onResult := func(path string, info fs.FileInfo) {
		log.Debugf("match: %s, dir=%v", path, info.IsDir())
		if info.IsDir() {
			return
		}
		fmt.Println(path)
	}
	s.execute(onResult)
}

type search struct {
	startPaths   []string
	fileMatchers []matcher
	pathMatchers []matcher
}

type searchResult string

func newSearchFromArgs(paths, fileFilters, fileNegFilters, pathFilters, pathNegFilters []string) (search, error) {
	s := search{}
	for _, fileFilterString := range fileFilters {
		pat, err := newRegexDotMatcher(fileFilterString)
		if err != nil {
			return s, err
		}
		s.fileMatchers = append(s.fileMatchers, pat)
	}
	for _, fileNegFilterString := range fileNegFilters {
		pat, err := newRegexDotMatcher(fileNegFilterString)
		if err != nil {
			return s, err
		}
		s.fileMatchers = append(s.fileMatchers, pat.negative())
	}
	for _, pathFilterString := range pathFilters {
		pat, err := newRegexDotMatcher(pathFilterString)
		if err != nil {
			return s, err
		}
		s.pathMatchers = append(s.pathMatchers, pat)
	}
	for _, pathNegFilterString := range pathNegFilters {
		pat, err := newRegexDotMatcher(pathNegFilterString)
		if err != nil {
			return s, err
		}
		s.pathMatchers = append(s.pathMatchers, pat.negative())
	}
	s.startPaths = append(s.startPaths, paths...)
	if len(s.startPaths) == 0 {
		s.startPaths = append(s.startPaths, ".")
	}
	return s, nil
}

func (s search) execute(onResult func(string, fs.FileInfo)) error {
	visitedPaths := make(map[string]bool)
	walkFunc := func(path string, info fs.FileInfo, err error) error {
		if visited := visitedPaths[path]; visited {
			log.Debugf("walk: already visited: %s", path)
			return nil
		}
		visitedPaths[path] = true
		log.Debugf("walk: %s", path)
		if err != nil {
			log.Printf("error for %s: %s", path, err)
			return nil
		}
		if !s.pathMatchesPatterns(path, info) {
			return nil
		}
		onResult(path, info)
		return nil
	}

	for _, root := range s.startPaths {
		log.Debugf("walk starting at: %s", root)
		if err := filepath.Walk(root, walkFunc); err != nil {
			return err
		}
	}
	return nil
}

func (s search) pathMatchesPatterns(path_ string, info fs.FileInfo) bool {
	pathLower := strings.ToLower(path_)

	// path must match all the path search patterns
	for _, pat := range s.pathMatchers {
		log.Debugf("check %s on %s", pat, pathLower)
		if !pat.match(pathLower) {
			log.Debugf("skip path %s because does not match %s", path_, pat)
			return false
		}
	}

	// content must match all the content patterns
	if info.IsDir() {
		return true
	}

	filename := path.Base(pathLower)
	// file name should match all the file search patterns
	for _, pat := range s.fileMatchers {
		if !pat.match(filename) {
			log.Debugf("skip file %s because does not match %s", filename, pat)
			return false
		}
	}

	return true
}

func (s search) String() string {
	return fmt.Sprintf("dirs: %+v, files: %+v, paths: %+v", s.startPaths, s.fileMatchers, s.pathMatchers)
}

type matcher interface {
	match(s string) bool
}

type regexDotMatcher struct {
	original string
	re       *regexp.Regexp
}

func newRegexDotMatcher(s string) (regexDotMatcher, error) {
	parts := strings.Split(s, "..")
	quoted := []string{}
	for _, part := range parts {
		quoted = append(quoted, regexp.QuoteMeta(part))
	}
	fullPattern := strings.Join(quoted, ".*?")
	fullPattern = "^" + fullPattern + "$"
	re, err := regexp.Compile(fullPattern)
	return regexDotMatcher{original: s, re: re}, err
}

func (mat regexDotMatcher) String() string {
	return fmt.Sprintf("%s /%s/", mat.original, mat.re)
}

func (mat regexDotMatcher) match(s string) bool {
	return mat.re.MatchString(s)
}

func (mat regexDotMatcher) negative() negRegexDotMatcher {
	return negRegexDotMatcher{original: mat}
}

type negRegexDotMatcher struct {
	original regexDotMatcher
}

func (mat negRegexDotMatcher) String() string {
	return fmt.Sprintf("not(%s)", mat.original)
}

func (mat negRegexDotMatcher) match(s string) bool {
	return !mat.original.match(s)
}
