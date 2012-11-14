package main


import (
	"path/filepath"
	"log"
	"sort"
	"github.com/rliebling/codesearch/index"
	"os"
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

func createIndex(args... string) {
	dirs_to_index := preparePaths(args)
	ix := index.Create(*indexFilename)
	ix.Verbose = *verboseFlag
	ix.AddPaths(dirs_to_index)
	for _, arg := range dirs_to_index {
		log.Printf("index %s", arg)
		filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
			if _, elem := filepath.Split(path); elem != "" {
				// Skip various temporary or "hidden" files or directories.
				if elem[0] == '.' || elem[0] == '#' || elem[0] == '~' || elem[len(elem)-1] == '~' {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
			if err != nil {
				log.Printf("%s: %s", path, err)
				return nil
			}
			if info != nil && info.Mode()&os.ModeType == 0 {
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

