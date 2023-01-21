package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	s, err := newSearchFromArgs(os.Args)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}
	onResult := func(r searchResult) {
		print(r)
	}

	s.execute(onResult)
}

type search struct {
	root                  string
	contentSearchPatterns []string
	fileSearchPatterns    []dotSearchPattern
	pathSearchPatterns    []dotSearchPattern
}

type searchResult string

const (
	fileSearchPrefix = "f="
	pathSearchPrefix = "p="
)

func newSearchFromArgs(args []string) (search, error) {
	s := search{root: "."} // todo how to set root in cli?
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
		} else {
			// content search pattern
			pat, err := contentSearchPatternFromArg(arg)
			if err != nil {
				return s, err
			}
			s.contentSearchPatterns = append(s.contentSearchPatterns, pat)
		}
	}
	return s, nil
}

func fileSearchPatternFromArg(arg string) (dotSearchPattern, error) {
	trimmed := strings.TrimPrefix(arg, fileSearchPrefix)
	if trimmed == arg {
		return dotSearchPattern{}, fmt.Errorf("file prefix missing for %s", arg)
	}
	return dotSearchPatternFromString(trimmed)
}

func pathSearchPatternFromArg(arg string) (dotSearchPattern, error) {
	trimmed := strings.TrimPrefix(arg, pathSearchPrefix)
	if trimmed == arg {
		return dotSearchPattern{}, fmt.Errorf("path prefix missing for %s", arg)
	}
	return dotSearchPatternFromString(trimmed)
}

func contentSearchPatternFromArg(arg string) (string, error) {
	// todo
	return arg, nil
}

func (s search) execute(onResult func(searchResult)) error {
	walkFunc := func(path string, info fs.FileInfo, err error) error {
		if !s.shouldContinueWithPath(path) {
			return nil
		}
		if err != nil {
			log.Printf("error for %s: %s", path, err)
			return nil
		}
	}
	return filepath.Walk(s.root, walkFunc)
}

func (s search) shouldContinueWithPath(path_ string) bool {
	filename := path.Base(path_)
	for _, pat := range s.fileSearchPatterns {
		if !pat.matches(filename) {
			return false
		}
	}
	for _, pat := range s.pathSearchPatterns {
		if !pat.matches(path_) {
			return false
		}
	}
	return true
}

type dotSearchPattern struct {
	original string
	re       *regexp.Regexp
}

func dotSearchPatternFromString(s string) (dotSearchPattern, error) {
	parts := strings.Split(s, "..")
	quoted := []string{}
	for _, part := range parts {
		quoted = append(quoted, regexp.QuoteMeta(part))
	}
	fullPattern := "" + strings.Join(quoted, ".*?") + ""
	re, err := regexp.Compile(fullPattern)
	return dotSearchPattern{original: s, re: re}, err
}

func (pat dotSearchPattern) matches(s string) bool {
	return pat.re.MatchString(s)
}
