package main

import (
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/rliebling/codesearch/index"
	"github.com/rliebling/gitignorer"
)

func preparePaths(args []string) []string {
	for i, arg := range args {
		a, err := filepath.Abs(arg)
		if err != nil {
			log.Printf("%s: %s", arg, err)
			args[i] = ""
			continue
		}
		args[i] = a
	}
	sort.Strings(args)

	for len(args) > 0 && args[0] == "" {
		args = args[1:]
	}
	return args
}

func createIndex(args ...string) {
	dirs_to_index := preparePaths(args)
	filter, _ := gitignorer.NewFilter()
	ix := index.Create(*indexFilename)
	ix.Verbose = *verboseFlag
	ix.AddPaths(dirs_to_index)
	for _, arg := range dirs_to_index {
		log.Printf("index %s", arg)
		filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Printf("%s: %s", path, err)
				return nil
			}

			if filter.Match(path) {
				return filepath.SkipDir
			}

			if info != nil && isRegularFile(info) {
				ix.AddFile(path)
			}
			return nil
		})
	}
	log.Printf("flush index")
	ix.Flush()
	ix.Close()

	log.Printf("done")
	return
}

func isRegularFile(info os.FileInfo) bool {
	return info.Mode()&os.ModeType == 0
}
