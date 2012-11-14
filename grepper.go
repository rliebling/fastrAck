package main

import (
	"bytes"
	"fmt"
	. "github.com/rliebling/codesearch/regexp"
	"io"
	"os"
)

type Grepper struct {
	Regexp        *Regexp // regexp to search for
	MatchCallback func(name string)
	LineCallback  func(name, line string, line_number int)
	CountCallback func(name string, count int)
	buf           []byte
}

func (g *Grepper) File(name string) {
	f, err := os.Open(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		fmt.Println("ERROR", err)
		return
	}
	defer f.Close()
	g.Reader(f, name)
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

func (g *Grepper) Reader(r io.Reader, name string) {
	if g.buf == nil {
		g.buf = make([]byte, 1<<20)
	}
	var (
		buf             = g.buf[:0]
		lineno          = 1
		beginText       = true
		endText         = false
		only_match_file = g.LineCallback == nil && g.CountCallback == nil
		count           = 0
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
			if g.MatchCallback != nil {
				g.MatchCallback(name)
			}
			if only_match_file {
				return
			}

			lineStart := bytes.LastIndex(buf[chunkStart:m1], nl) + 1 + chunkStart
			lineEnd := m1 + 1
			if lineEnd > end {
				lineEnd = end
			}
			lineno += countNL(buf[chunkStart:lineStart])
			line := buf[lineStart:lineEnd]

			count++
			if g.LineCallback != nil {
				g.LineCallback(name, string(line), lineno)
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
	if g.CountCallback != nil && count > 0 {
		g.CountCallback(name, count)
	}
	return
}
