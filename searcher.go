package main

import (
  //"github.com/howeyc/fsnotify"
  "github.com/rliebling/ackinaflash/dir_watcher"
	"flag"
  "fmt"
  //"os"
)

func main() {
	flag.Parse()
  fmt.Printf("Hello %v\n", flag.Args())
	watcher, err := dir_watcher.Watch(flag.Args())
	if err != nil {
		panic("Failed to watch successfully " + err.Error())
	} else {
		fmt.Println("Success")
	}
	defer watcher.Close()
}

