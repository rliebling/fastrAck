package dir_watcher

import (
	"errors"
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

func Watch(dirnames []string) (w Watcher, err error) {
	dirs_to_watch := make(chan string, 50)
	go func() {
		for _, dir := range dirnames {
			dirs_to_watch <- dir
		}
	}()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		errors.New("failed initializing watcher " + err.Error())
	}
	//defer watcher.Close()

	watchDirs(watcher, dirs_to_watch)
	w = Watcher{Events: watcher.Event}
	return
}

func watchDirs(watcher *fsnotify.Watcher, dirs_to_watch chan string) {
	for {
		select {
		case dirname := <- dirs_to_watch:
			processDirname(watcher, dirname, dirs_to_watch)
		default:
			fmt.Println("DONE")
			return
		}
	}
}

func processDirname(watcher *fsnotify.Watcher, dirname string, dirs_to_watch chan string) {
	err := watcher.Watch(dirname)
	if err != nil {
		panic("failed initiating watch on " + dirname + " " + err.Error())
	}
	fmt.Fprintln(os.Stderr, "Now watching", dirname)

	dir, err := os.Open(dirname)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not open", dirname, "got error", err)
	}
	subdirs, err := dir.Readdirnames(-1)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not readdirnames", dirname, "got error", err)
	}
	for _, subdirname := range subdirs {
		dirs_to_watch <- dirname + string(os.PathSeparator) + subdirname
	}
}

func (w Watcher) Close() {
	fmt.Println("Closing")
	go func() {
		for e:= range w.watcher.Error {
			fmt.Println("Error " + e.Error())
		}
	}()
	w.watcher.Close()
	fmt.Println("after Closing")
}
