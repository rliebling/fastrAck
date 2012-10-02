package main

import (
	"bytes"
	"os"
	"io"
	"fmt"
	. "github.com/rliebling/codesearch/regexp"
)


type MyGrep struct {
	Regexp *Regexp   // regexp to search for
	Stderr io.Writer // error target

	Match bool

	buf []byte
}
type MyGrepResult struct {
	IsMatch bool
	Count int
	Matches []MyGrepMatch
}

type MyGrepRequest int
const (
	NeedMatches MyGrepRequest = iota
	NeedFileOnly
	NeedCountOnly
)

type MyGrepMatch struct {
	Line string
	LineNumber int
  MatchStartIndex int
	MatchEndIndex int
}

func (g *MyGrep) File(name string, data_needed MyGrepRequest) ( *MyGrepResult) {
	f, err := os.Open(name)
	if err != nil {
		fmt.Fprintf(g.Stderr, "%s\n", err)
		fmt.Println("ERROR", err)
		return new( MyGrepResult)
	}
	defer f.Close()
	return g.Reader(f, data_needed)
}

var nl = []byte{'\n'}

func countNL(b []byte) int {
	n := 0
	for {
		i := bytes.IndexByte(b, '\n')
		if i < 0 {
			break
		}
		n++
		b = b[i+1:]
	}
	return n
}

func (g *MyGrep) Reader(r io.Reader, data_needed MyGrepRequest) (result *MyGrepResult) {
	result = new(MyGrepResult)
	if g.buf == nil {
		g.buf = make([]byte, 1<<20)
	}
	var (
		buf        = g.buf[:0]
		lineno     = 1
		beginText  = true
		endText    = false
	)
	for {
		n, err := io.ReadFull(r, buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+n]
		end := len(buf)
		if err == nil {
			end = bytes.LastIndex(buf, nl) + 1
		} else {
			endText = true
		}
		chunkStart := 0
		for chunkStart < end {
			m1 := g.Regexp.Match(buf[chunkStart:end], beginText, endText) + chunkStart
			beginText = false
			if m1 < chunkStart {
				break
			}
			g.Match = true
			result.IsMatch = true
			if data_needed == NeedFileOnly {
				return
			}

			lineStart := bytes.LastIndex(buf[chunkStart:m1], nl) + 1 + chunkStart
			lineEnd := m1 + 1
			if lineEnd > end {
				lineEnd = end
			}
			lineno += countNL(buf[chunkStart:lineStart])
			line := buf[lineStart:lineEnd]

			result.Count++
			if data_needed == NeedMatches {
				result.Matches = append(result.Matches, MyGrepMatch{Line: string(line), LineNumber: lineno, MatchStartIndex:0 })
			}
			lineno++
			chunkStart = lineEnd
		}
		if err == nil {
			lineno += countNL(buf[chunkStart:end])
		}
		n = copy(buf, buf[end:])
		buf = buf[:n]
		if len(buf) == 0 && err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
			break
		}
	}
	return
}
