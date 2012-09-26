package main

import (
  //"github.com/howeyc/fsnotify"
	"github.com/rliebling/codesearch/index"
  "github.com/rliebling/fastrAck/dir_watcher"
	"path/filepath"
	"flag"
  "fmt"
	"io"
  "os"
//	"strings"
	"time"
)

type fileset map[string]bool

func main() {
	flag.Parse()
	curdir,_ := filepath.Abs(".")
  fmt.Printf("Hello %v %v\n", flag.Args(), curdir)
	watcher, err := dir_watcher.Watch(flag.Args(), func(dirname string)bool {
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
			case ev := <- watcher.Events:
				finfo, err := os.Stat(ev.Name)
				if err != nil || finfo.IsDir() {
					continue
				}

				if !ev.IsDelete() {
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
			}
		}
	}()

	for {
		select {
		case paths := <-trigger_reindex:
			path_array := make([]string, len(paths))
			i := 0
			for p,_ := range paths {
				path_array[i] = p
				i++
			}
			reindex(path_array, curdir)
		case <-search_request:
		}

	}
}

func reindex(paths []string, curdir string) {
	fmt.Printf("Reindexing %v\n", paths)
	master := index.File()
	file := master + "~"
	ix := index.Create(file)
	//ix.AddPaths([]string{curdir})
	//ix.AddPaths([]string{"c:\\Users\\rich\\workspace\\sitm"})
	for _, p := range paths {
		if _, elem := filepath.Split(p); elem != "" {
			// Skip various temporary or "hidden" files or directories.
			if elem[0] == '.' || elem[0] == '#' || elem[0] == '~' || elem[len(elem)-1] == '~' {
				fmt.Println("Skipping " + p)
				continue
			}
			if _,err := os.Stat(p); err!=nil {
				fmt.Println("Skipping (err stat'ing )" + p)
				continue
			}
		}
		//fmt.Println("AddFile " , p)
		ix.AddPaths([]string{p})
		ix.AddFile(p)
	}
	ix.Flush()
	ix.Close()

	index.Merge(file+"~", master, file)
	_, err := copyFile(master, file+"~")
	if err != nil {
		panic("copy: " + err.Error())
	}
	//err = os.Remove(file)
	if err != nil {
		panic("Remove: " + err.Error())
	}
	fmt.Println("Updated file " + master)
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



//RLIEBLING3








