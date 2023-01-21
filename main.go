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
	var fileFilters stringArgs
	var pathFilters stringArgs
	var flagDebug bool
	flag.BoolVar(&flagDebug, "v", false, "verbose debug mode")
	flag.Var(&fileFilters, "f", "file filters")
	flag.Var(&pathFilters, "p", "path filters")
	flag.Parse()

	log.Debug = flagDebug

	s, err := newSearchFromArgs(flag.Args(), fileFilters, pathFilters)
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
	startPaths         []string
	fileSearchPatterns []dotSearchPattern
	pathSearchPatterns []dotSearchPattern
}

type searchResult string

func newSearchFromArgs(paths, fileFilters, pathFilters []string) (search, error) {
	s := search{}
	for _, fileFilterString := range fileFilters {
		pat, err := dotSearchPatternFromString(fileFilterString, true)
		if err != nil {
			return s, err
		}
		s.fileSearchPatterns = append(s.fileSearchPatterns, pat)
	}
	for _, pathFilterString := range pathFilters {
		pat, err := dotSearchPatternFromString(pathFilterString, true)
		if err != nil {
			return s, err
		}
		s.pathSearchPatterns = append(s.pathSearchPatterns, pat)
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
	for _, pat := range s.pathSearchPatterns {
		log.Debugf("check %s on %s", pat.re, pathLower)
		if !pat.re.MatchString(pathLower) {
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
	for _, pat := range s.fileSearchPatterns {
		if !pat.re.MatchString(filename) {
			log.Debugf("skip file %s because does not match %s", filename, pat)
			return false
		}
	}

	return true
}

func (s search) String() string {
	return fmt.Sprintf("dirs: %+v, files: %+v, paths: %+v", s.startPaths, s.fileSearchPatterns, s.pathSearchPatterns)
}

type dotSearchPattern struct {
	original string
	re       *regexp.Regexp
}

func dotSearchPatternFromString(s string, matchWholeContent bool) (dotSearchPattern, error) {
	parts := strings.Split(s, "..")
	quoted := []string{}
	for _, part := range parts {
		quoted = append(quoted, regexp.QuoteMeta(part))
	}
	fullPattern := strings.Join(quoted, ".*?")
	if matchWholeContent {
		fullPattern = "^" + fullPattern + "$"
	}
	re, err := regexp.Compile(fullPattern)
	return dotSearchPattern{original: s, re: re}, err
}

func (pat dotSearchPattern) String() string {
	return fmt.Sprintf("%s /%s/", pat.original, pat.re)
}
