package main

import (
  //"github.com/howeyc/fsnotify"
	"github.com/rliebling/codesearch/index"
	"github.com/rliebling/codesearch/regexp"
  "github.com/rliebling/fastrAck/dir_watcher"
	"path/filepath"
	"flag"
  "fmt"
	"io"
	"log"
  "os"
	"sort"
//	"strings"
	"time"
)

type fileset map[string]bool

var (
	fileFilterFlag = flag.String("f", "", "search only files with names matching this regexp")
	fileExclusionFlag = flag.String("F", "", "search excluding files with names matching this regexp")
	iFlag       = flag.Bool("i", false, "case-insensitive search")
	listFlag    = flag.Bool("list", false, "list indexed paths and exit")
	indexFlag   = flag.Bool("index", false, "create index")
	watchFlag   = flag.Bool("watch", false, "watch for changes")
	indexFilename = flag.String("file", "./.cindex", "index filename")
	verboseFlag = flag.Bool("verbose", false, "print extra information")
	cpuProfile  = flag.String("cpuprofile", "", "write cpu profile to this file")
)

func main() {
	flag.Parse()
	if *watchFlag {
		createIndex(".")
		watch(".")
	} else if *indexFlag {
		createIndex(".")
	} else {
		search(flag.Args()...)
	}
}

func search(args... string) {
	g := regexp.Grep{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		N: true,
	}

	pat := "(?m)" + args[0]
	if *iFlag {
		pat = "(?i)" + pat
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		log.Fatal(err)
	}
	g.Regexp = re
	var fre, fexclusion_re *regexp.Regexp
	if *fileFilterFlag != "" {
		fre, err = regexp.Compile(*fileFilterFlag)
		if err != nil {
			log.Fatal(err)
		}
	}
	if *fileExclusionFlag != "" {
		fexclusion_re, err = regexp.Compile(*fileExclusionFlag)
		if err != nil {
			log.Fatal(err)
		}
	}
	q := index.RegexpQuery(re.Syntax)
	if *verboseFlag {
		log.Printf("query: %s\n", q)
	}

	ix := index.Open(*indexFilename)
	ix.Verbose = *verboseFlag
	var post []uint32
	post = ix.PostingQuery(q)

	if *verboseFlag {
		log.Printf("post query identified %d possible files\n", len(post))
	}

	if fre != nil || fexclusion_re != nil {
		fnames := make([]uint32, 0, len(post))

		for _, fileid := range post {
			name := ix.Name(fileid)
			if fre!=nil && fre.MatchString(name, true, true) < 0 {
				continue
			}
			if fexclusion_re!=nil && fexclusion_re.MatchString(name, true, true) >= 0 {
				continue
			}
			fnames = append(fnames, fileid)
		}

		if *verboseFlag {
			log.Printf("filename regexp matched %d files\n", len(fnames))
		}
		post = fnames
	}

	for _, fileid := range post {
		name := ix.Name(fileid)
		g.File(name)
	}

	//matches = g.Match
}

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

	log.Printf("done")
	return
}

func watch(args... string) {
	curdir,_ := filepath.Abs(".")
  fmt.Printf("Hello %v %v\n", args, curdir)
	watcher, err := dir_watcher.Watch(args, func(dirname string)bool {
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

func skip_reindex(name string) bool {
	if _, elem := filepath.Split(name); elem != "" {
		// Skip various temporary or "hidden" files or directories.
		if elem[0] == '.' || elem[0] == '#' || elem[0] == '~' || elem[len(elem)-1] == '~' {
			fmt.Println("Skipping " + name)
			return true
		}
		if _,err := os.Stat(name); err!=nil {
			fmt.Println("Skipping (err stat'ing )" + name)
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
	//ix.AddPaths([]string{curdir})
	//ix.AddPaths([]string{"c:\\Users\\rich\\workspace\\sitm"})
	added_files := false
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
		fmt.Println("AddFile " , p)
		ix.AddPaths([]string{p})
		ix.AddFile(p)
		added_files = true
	}
	ix.Flush()
	ix.Close()

	if !added_files {
		os.Remove(file)
		return
	}

	index.Merge(file+"~", master, file)
	_, err := copyFile(master, file+"~")
	if err != nil {
		panic("copy: " + err.Error())
	}
	err = os.Remove(file)
	if err != nil {
		panic("Remove: " + err.Error())
	}
	err = os.Remove(file+"~")
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








