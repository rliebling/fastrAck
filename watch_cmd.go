package main

import (
	"fmt"
	"github.com/rliebling/codesearch/index"
	"github.com/rliebling/fastrAck/dir_watcher"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"
)

func watch(args ...string) {
	curdir, _ := filepath.Abs(".")
	fmt.Printf("Hello %v %v\n", args, curdir)
	watcher, err := dir_watcher.Watch(args, func(dirname string) bool {
		_, elem := filepath.Split(dirname)
		// Skip various temporary or "hidden" files or directories.
		return !(elem[0] == '.' || elem[0] == '#' || elem[0] == '~' || elem[len(elem)-1] == '~')
	})

	if err != nil {
		panic("Failed to watch successfully " + err.Error())
	}
	//defer watcher.Close()

	//for ev := range watcher.Events {
	waitPeriod := 1000 * time.Hour
	to_reindex := make(fileset, 100)
	trigger_reindex := make(chan fileset, 3)
	search_request := make(chan string, 3)
	go func() {
		for {
			select {
			case ev := <-watcher.Events:
				finfo, err := os.Stat(ev.Name)
				if err != nil || finfo.IsDir() {
					continue
				}

				if !ev.IsDelete() && !skip_reindex(ev.Name) { // need to handle delete by NOT reindexing this file
					//fmt.Printf("EVENT: %s %#v\n" , ev.Name ,  ev)
					//name := strings.Replace(ev.Name, "/", "\\", -1)
					to_reindex[ev.Name] = true
					waitPeriod = 10 * time.Second
				}
			case <-time.After(waitPeriod):
				//fmt.Printf("wait awoke %v\n", to_reindex)
				trigger_reindex <- to_reindex
				to_reindex = make(fileset, 100)
				waitPeriod = 1000 * time.Hour
				prof_file, err := os.Create("profile.prof")
				if err != nil {
					log.Println("**************************************")
					log.Println("No profile because " + err.Error())
					log.Println("**************************************")
				} else {
					log.Println("**************************************")
					log.Println("Wrote profile")
					log.Println("**************************************")
					pprof.WriteHeapProfile(prof_file)
					prof_file.Close()
				}
			}
		}
	}()

	for {
		select {
		case paths := <-trigger_reindex:
			path_array := make([]string, len(paths))
			i := 0
			for p, _ := range paths {
				path_array[i] = p
				i++
			}
			reindex(path_array, curdir)
		case <-search_request:
		}

	}
}

func skip_reindex(name string) bool {
	if _, elem := filepath.Split(name); elem != "" {
		// Skip various temporary or "hidden" files or directories.
		if elem[0] == '.' || elem[0] == '#' || elem[0] == '~' || elem[len(elem)-1] == '~' {
			log.Println("Skipping " + name)
			return true
		}
		if _, err := os.Stat(name); err != nil {
			log.Println("Skipping (err stat'ing )" + name)
			return true
		}
	}
	return false
}

func reindex(paths []string, curdir string) {
	paths = preparePaths(paths)
	log.Printf("Reindexing %v\n", paths)
	master := *indexFilename
	log.Println("Master is ", master)
	file := master + "~"
	ix := index.Create(file)
	defer cleanupFile(file)

	//ix.AddPaths([]string{curdir})
	//ix.AddPaths([]string{"c:\\Users\\rich\\workspace\\sitm"})
	added_files := false
	for _, p := range paths {
		if _, elem := filepath.Split(p); elem != "" {
			// Skip various temporary or "hidden" files or directories.
			if elem[0] == '.' || elem[0] == '#' || elem[0] == '~' || elem[len(elem)-1] == '~' {
				log.Println("reindex skipping " + p)
				continue
			}
			if _, err := os.Stat(p); err != nil {
				log.Println("reindex skipping (err stat'ing )" + p)
				continue
			}
		}
		log.Println("AddFile ", p)
		ix.AddPaths([]string{p})
		ix.AddFile(p)
		added_files = true
	}
	ix.Flush()
	ix.Close()

	if !added_files {
		return
	}

	index.Merge(file+"~", master, file)
	defer cleanupFile(file + "~")

	_, err := copyFile(master, file+"~")
	if err != nil {
		panic("copy: " + err.Error())
	}
	log.Println("Updated file " + master)
}

func cleanupFile(file string) {
	err := os.Remove(file)
	if err != nil {
		panic("Remove: " + err.Error())
	}
}

func copyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()

	dst, err := os.Create(dstName)
	if err != nil {
		return
	}
	defer dst.Close()

	return io.Copy(dst, src)
}
