package dir_watcher

import (
	"errors"
	"path/filepath"
	"fmt"
  "github.com/howeyc/fsnotify"
  "os"
)

/*
type FileEvent interface {
  Name() string
  IsCreate() bool
	IsDelete() bool
	IsModify() bool
	IsRename() bool
}
*/

type Watcher struct {
	Events chan *fsnotify.FileEvent
	watcher fsnotify.Watcher
}

func Watch(dirnames []string, dir_filter func(string)bool) (w Watcher, err error) {
	dirs_to_watch := make(chan []string, 50)
	dirs_to_watch <- dirnames

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		errors.New("failed initializing watcher " + err.Error())
	}
	//defer watcher.Close()

	watchDirs(watcher, dirs_to_watch, dir_filter)
	w = Watcher{Events: pathNormalizer(watcher.Event)}
	return
}

func watchDirs(watcher *fsnotify.Watcher, dirs_to_watch chan []string, dir_filter func(string)bool) {
	for {
		select {
		case dirnames := <- dirs_to_watch:
			subdirs := make([]string, 0, 5)
			for _, d := range dirnames {
				for _,sd := range processDirname(watcher, d) {
					if dir_filter(sd) {
						subdirs = append(subdirs, sd)
					} else {
						fmt.Println("Filtering out", sd)
					}
				}
			}
			if len(subdirs) > 0 {
				//fmt.Printf("Adding to queue: %v\n" , subdirs)
				dirs_to_watch <- subdirs
			}
		default:
			fmt.Println("DONE")
			return
		}
	}
}

func processDirname(watcher *fsnotify.Watcher, dirname string) (subdirs []string) {
	//fmt.Printf("processDirname %v %d %v %x\n", dirname, len(dirname), subdirs, len(subdirs))
	abs_path, _ := filepath.Abs(dirname)
	err := watcher.Watch(abs_path)
	if err != nil {
		panic("failed initiating watch on " + dirname + " " + err.Error())
	}
	//fmt.Fprintln(os.Stderr, "Now watching", dirname)

	dir, err := os.Open(dirname)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not open", dirname, "got error", err)
	}
	defer dir.Close()

	dir_entries, err := dir.Readdir(-1)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not readdir", dirname, "got error", err)
	}

	for _, entry := range dir_entries {
		if !entry.IsDir() {
			continue
		}

		abs_name, _ := filepath.Abs(dirname + string(os.PathSeparator) + entry.Name())
		//fmt.Println("ABS: ", dirname, string(os.PathSeparator), entry.Name(), abs_name)
		subdirs = append(subdirs, abs_name)
	}

	return
}

func (w Watcher) Close() {
	//fmt.Println("Closing")
	go func() {
		for e:= range w.watcher.Error {
			fmt.Println("Error " + e.Error())
		}
	}()
	w.watcher.Close()
}

func pathNormalizer(in chan *fsnotify.FileEvent) chan *fsnotify.FileEvent {
	out := make(chan *fsnotify.FileEvent, 10)
	go func() {
		for {
			e := <-in
			norm_name, err := filepath.Abs(e.Name)
			if err != nil {
				panic("Error: failed normalizing " + e.Name)
			}
			e.Name = norm_name
			out<- e
		}
	}()
	return out
}
