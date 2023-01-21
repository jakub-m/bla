package main

import (
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var log = betterlog{}

const helpString = `
Usage:

	bla <paths> \
		f=<file match pattern> \
	    p=<path match pattern> \
		c=<content match pattern>

Where the match pattern is a set of literals separted by two dots "..",
like: ..foo..bar.. or foo..bar

For files and paths, the pattern matches whole file or path. For content, the
pattern matches any part of the content (.. are added implicitly at the
beginning and the end of the pattern.)

Path and filenames are case-insensitive and the patterns must be lower-case.
Content is case sensitive unless one passes -i flag.
`

func main() {
	var flagDebug bool
	flag.BoolVar(&flagDebug, "v", false, "verbose debug mode")
	var flagContentCaseInsensitive bool
	flag.BoolVar(&flagContentCaseInsensitive, "i", false, "case insensitive content matches (slower)")
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Usage = func() {
		fmt.Println(strings.Trim(helpString, " \n"))
		fmt.Println()
		flag.PrintDefaults()
	}
	flag.Parse()

	log.Debug = flagDebug

	s, err := newSearchFromArgs(flag.Args())
	if err != nil {
		log.Fatalf("Error: %s", err)
	}
	log.Debugf("search: %s", s)
	onResult := func(path string, info fs.FileInfo) {
		log.Debugf("matched: %s ,dir=%v", path, info.IsDir())
		if info.IsDir() {
			return
		}
		fmt.Println(path)
	}
	s.execute(onResult, flagContentCaseInsensitive)
}

type search struct {
	startPaths            []string
	contentSearchPatterns []dotSearchPattern
	fileSearchPatterns    []dotSearchPattern
	pathSearchPatterns    []dotSearchPattern
}

type searchResult string

const (
	contentSearchPrefix = "c="
	fileSearchPrefix    = "f="
	pathSearchPrefix    = "p="
)

func newSearchFromArgs(args []string) (search, error) {
	s := search{}
	for _, arg := range args {
		if strings.HasPrefix(arg, fileSearchPrefix) {
			pat, err := fileSearchPatternFromArg(arg)
			if err != nil {
				return s, err
			}
			s.fileSearchPatterns = append(s.fileSearchPatterns, pat)
			// file name search pattern
		} else if strings.HasPrefix(arg, pathSearchPrefix) {
			// path search pattern
			pat, err := pathSearchPatternFromArg(arg)
			if err != nil {
				return s, err
			}
			s.pathSearchPatterns = append(s.pathSearchPatterns, pat)
		} else if strings.HasPrefix(arg, contentSearchPrefix) {
			// content search pattern
			pat, err := contentSearchPatternFromArg(arg)
			if err != nil {
				return s, err
			}
			s.contentSearchPatterns = append(s.contentSearchPatterns, pat)
		} else {
			s.startPaths = append(s.startPaths, arg)
		}
	}
	if len(s.startPaths) == 0 {
		s.startPaths = append(s.startPaths, ".")
	}
	return s, nil
}

func fileSearchPatternFromArg(arg string) (dotSearchPattern, error) {
	trimmed := strings.TrimPrefix(arg, fileSearchPrefix)
	if trimmed == arg {
		return dotSearchPattern{}, fmt.Errorf("file prefix missing for %s", arg)
	}
	return dotSearchPatternFromString(trimmed, true)
}

func pathSearchPatternFromArg(arg string) (dotSearchPattern, error) {
	trimmed := strings.TrimPrefix(arg, pathSearchPrefix)
	if trimmed == arg {
		return dotSearchPattern{}, fmt.Errorf("path prefix missing for %s", arg)
	}
	return dotSearchPatternFromString(trimmed, true)
}

func contentSearchPatternFromArg(arg string) (dotSearchPattern, error) {
	trimmed := strings.TrimPrefix(arg, contentSearchPrefix)
	if trimmed == arg {
		return dotSearchPattern{}, fmt.Errorf("content prefix missing for %s", arg)
	}
	// for content, we are not interested in full text matches but submatches by default
	return dotSearchPatternFromString(trimmed, false)
}

func (s search) execute(onResult func(string, fs.FileInfo), contentCaseInsensitive bool) error {
	visitedPaths := make(map[string]bool)
	walkFunc := func(path string, info fs.FileInfo, err error) error {
		if visited := visitedPaths[path]; visited {
			log.Debugf("walk: already visited: %s", path)
			return nil
		}
		visitedPaths[path] = true
		log.Debugf("walk: %s", path)
		if !s.pathMatchesPatterns(path, info, contentCaseInsensitive) {
			return nil
		}
		if err != nil {
			log.Printf("error for %s: %s", path, err)
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

func (s search) pathMatchesPatterns(path_ string, info fs.FileInfo, contentCaseInsensitive bool) bool {
	// Adding (?i) to the unrelying regex makes the significantly slower.
	pathLower := strings.ToLower(path_)

	// path must match all the path search patterns
	for _, pat := range s.pathSearchPatterns {
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

	if len(s.contentSearchPatterns) > 0 {
		// do not open file if no content search patterns
		contentBytes, err := ioutil.ReadFile(path_)
		if err != nil {
			return false
		}
		content := string(contentBytes)
		if contentCaseInsensitive {
			content = strings.ToLower(content)
		}
		for _, pat := range s.contentSearchPatterns {
			// assume content is utf-8.
			if !pat.re.MatchString(content) {
				log.Debugf("skip content of %s because does not match %s", path_, pat)
				return false
			}
		}
	}

	return true
}

func (s search) String() string {
	return fmt.Sprintf("dirs: %+v, files: %+v, paths: %+v, content: %+v", s.startPaths, s.fileSearchPatterns, s.pathSearchPatterns, s.contentSearchPatterns)
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
