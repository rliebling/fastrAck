package main

import (
	//"github.com/howeyc/fsnotify"
	"flag"
	"fmt"
	"github.com/rliebling/codesearch/index"
	"github.com/rliebling/codesearch/regexp"
	"github.com/rliebling/terminal"
	"io"
	"log"
	"os"
	"os/exec"
	std_regexp "regexp"
	"strings"
	"syscall"
)

type fileset map[string]bool

var (
	fileFilterFlag    = flag.String("f", "", "search only files with names matching this regexp")
	fileExclusionFlag = flag.String("F", "", "search excluding files with names matching this regexp")
	iFlag             = flag.Bool("i", false, "case-insensitive search")
	nameOnlyFlag      = flag.Bool("l", false, "only print filenames that match")
	colorFlag         = flag.Bool("color", true, "show results with coloring")
	terminalFlag      = flag.Bool("terminal", true, "treat as if going to human reader")
	indexFlag         = flag.Bool("index", false, "create index")
	watchFlag         = flag.Bool("watch", false, "watch for changes")
	indexFilename     = flag.String("file", ".cindex", "index filename")
	verboseFlag       = flag.Bool("verbose", false, "print extra information")
	cpuProfile        = flag.String("cpuprofile", "", "write cpu profile to this file")
)

func main() {
	flag.Parse()
	if len(os.Args) < 2 {
		flag.PrintDefaults()
		return
	}
	if *watchFlag {
		createIndex(".")
		watch(".")
	} else if *indexFlag {
		createIndex(".")
	} else {
		search(flag.Args()...)
	}
}

func search(args ...string) {
	var stdout io.WriteCloser
	var err error

	is_terminal := *terminalFlag && terminal.IsTerminal(syscall.Stdout)
	if !is_terminal {
		*colorFlag = false
	}

	if *colorFlag && strings.HasPrefix(os.Getenv("OS"), "Windows") {
		cmd := exec.Command("ruby", "-rubygems", "-rwin32console", "-e", "puts STDIN.readlines")
		cmd.Stdout = os.Stdout
		stdout, err = cmd.StdinPipe()
		cmd.Start()
		defer cmd.Wait()
		defer stdout.Close()
	} else {
		stdout = os.Stdout
	}

	pat := "(?m)" + args[0]
	if *iFlag {
		pat = "(?i)" + pat
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		log.Fatal(err)
	}

	g := Grepper{}
	if is_terminal {
		if *colorFlag {
			g.MatchCallback = func(name string) {
				fmt.Fprintf(stdout, "\033[1;31m%s\033[0m\n", name)
			}
			if !*nameOnlyFlag {
				std_re, _ := std_regexp.Compile(pat)
				g.LineCallback = func(name, line string, line_number int) {
					// nuke EOL and wrap with coloring
					eol := len(line)
					if line[eol-1:eol] == "\n" {
						eol = eol - 1
					}
					result := std_re.ReplaceAllString(line[:eol], "\033[1;37m\033[1;41m$0\033[0m")
					fmt.Fprintf(stdout, "%d|\t%s\n", line_number, result)
				}
			}
		} else {
			g.MatchCallback = func(name string) {
				fmt.Fprintf(stdout, "%s\n", name)
			}
			if !*nameOnlyFlag {
				g.LineCallback = func(name, line string, line_number int) {
					fmt.Fprintf(stdout, "%d|\t%s\n", line_number, line[:len(line)-1])
				}
			}
		}
	} else {
		if *nameOnlyFlag {
			g.MatchCallback = func(name string) {
				fmt.Fprintf(stdout, "%s\n", name)
			}
		} else {
			g.LineCallback = func(name, line string, line_number int) {
				fmt.Fprintf(stdout, "%s:%d: %s\n", name, line_number, line[:len(line)-1])
			}
		}
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

	*indexFilename = findIndexFile(*indexFilename)
	if !exists(*indexFilename) {
		log.Fatalf("Could not find %s", *indexFilename)
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
			if fre != nil && fre.MatchString(name, true, true) < 0 {
				continue
			}
			if fexclusion_re != nil && fexclusion_re.MatchString(name, true, true) >= 0 {
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

func findIndexFile(indexFileName string) string {
	workingDirectory, _ := os.Getwd()
	searchDepth := strings.Count(workingDirectory, string(os.PathSeparator))

	searchPath := indexFileName
	for depth := 0; depth < searchDepth; depth++ {
		if exists(searchPath) {
			if *verboseFlag {
				log.Printf("Found .cindex in %s", searchPath)
			}
			return searchPath
		}
		searchPath = "../" + searchPath
	}

	return indexFileName
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}
