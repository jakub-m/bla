package main

import (
	"fmt"
	"log"
	"os"
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
	contentSearchPatterns []string
	fileSearchPatterns    []string
	pathSearchPatterns    []string
}

type searchResult string

const (
	fileSearchPrefix = "f="
	pathSearchPrefix = "p="
)

func newSearchFromArgs(args []string) (search, error) {
	s := search{}
	for _, arg := range args {
		if strings.HasPrefix(arg, fileSearchPrefix) {
			pat, err := fileSearchPatternFromArg(arg)
			if err != nil {
				return s, err
			}
			s.contentSearchPatterns = append(s.fileSearchPatterns, pat)
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

func fileSearchPatternFromArg(arg string) (string, error) {
	trimmed := strings.TrimPrefix(arg, fileSearchPrefix)
	if trimmed == arg {
		return "", fmt.Errorf("file prefix missing for %s", arg)
	}
	return trimmed, nil
}

func pathSearchPatternFromArg(arg string) (string, error) {
	trimmed := strings.TrimPrefix(arg, pathSearchPrefix)
	if trimmed == arg {
		return "", fmt.Errorf("path prefix missing for %s", arg)
	}
	return trimmed, nil
}

func contentSearchPatternFromArg(arg string) (string, error) {
	return arg, nil
}

func (s search) execute(onResult func(searchResult)) error {
}
