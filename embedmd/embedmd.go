// Copyright 2016 Google Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to writing, software distributed
// under the License is distributed on a "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.

// Package embedmd provides a single function, Process, that parses markdown
// searching for markdown comments.
//
// The format of an embedmd command is:
//
//     [embedmd]:# (pathOrURL language /start regexp/ /end regexp/)
//
// The embedded code will be extracted from the file at pathOrURL,
// which can either be a relative path to a file in the local file
// system (using always forward slashes as directory separator) or
// a url starting with http:// or https://.
// If the pathOrURL is a url the tool will fetch the content in that url.
// The embedded content starts at the first line that matches /start regexp/
// and finishes at the first line matching /end regexp/.
//
// Omitting the the second regular expression will embed only the piece of
// text that matches /regexp/:
//
//     [embedmd]:# (pathOrURL language /regexp/)
//
// To embed the whole line matching a regular expression you can use:
//
//     [embedmd]:# (pathOrURL language /.*regexp.*\n/)
//
// If you want to embed from a point to the end you should use:
//
//     [embedmd]:# (pathOrURL language /start regexp/ $)
//
// Finally you can embed a whole file by omitting both regular expressions:
//
//     [embedmd]:# (pathOrURL language)
//
// You can ommit the language in any of the previous commands, and the extension
// of the file will be used for the snippet syntax highlighting. Note that while
// this works Go files, since the file extension .go matches the name of the language
// go, this will fail with other files like .md whose language name is markdown.
//
//     [embedmd]:# (file.ext)
//
package embedmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Process reads markdown from the given io.Reader searching for an embedmd
// command. When a command is found, it is executed and the output is written
// into the given io.Writer with the rest of standard markdown.
func Process(out io.Writer, in io.Reader, opts ...Option) error {
	e := embedder{Fetcher: fetcher{}}
	for _, opt := range opts {
		opt.f(&e)
	}
	return process(out, in, e.runCommand)
}

// An Option provides a way to adapt the Process function to your needs.
type Option struct{ f func(*embedder) }

// WithBaseDir indicates that the given path should be used to resolve relative
// paths.
func WithBaseDir(path string) Option {
	return Option{func(e *embedder) { e.baseDir = path }}
}

// WithFetcher provides a custom Fetcher to be used whenever a path or url needs
// to be fetched.
func WithFetcher(c Fetcher) Option {
	return Option{func(e *embedder) { e.Fetcher = c }}
}

type embedder struct {
	Fetcher
	baseDir string
}

func (e *embedder) runCommand(w io.Writer, cmd *command) error {
	b, err := e.Fetch(e.baseDir, cmd.path)
	if err != nil {
		return fmt.Errorf("could not read %s: %v", cmd.path, err)
	}
	b, err = extract(b, cmd.sample)
	if err != nil {
		return fmt.Errorf("could not extract content from %s: %v", cmd.path, err)
	}

	if len(b) > 0 && b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}

	fmt.Fprintln(w, "```go")
	scanner := bufio.NewScanner(bytes.NewBuffer(b))
	var code []string
	for scanner.Scan() {
		t := scanner.Text()
		if strings.Contains(t, "START") {
			continue
		}
		if strings.Contains(t, "END") {
			continue
		}
		code = append(code, scanner.Text())
	}
	for _, c := range normalize(code) {
		fmt.Fprintln(w, c)
	}
	fmt.Fprintln(w, "```")
	return nil
}

func normalize(s []string) []string {
	indent := -1
loop:
	for i := 0; ; i++ {
		for _, line := range s {
			if line == "" {
				continue
			}
			if strings.LastIndex(line, "\t") != i {
				break loop
			}
			indent = i
		}
	}
	if indent == -1 {
		return s
	}
	for i, line := range s {
		if line == "" {
			continue
		}
		s[i] = line[indent+1:]
	}
	return s
}

func extract(b []byte, sample string) ([]byte, error) {
	start := "START " + sample
	end := "END " + sample
	match := func(s string) ([]int, error) {
		re, err := regexp.CompilePOSIX(s)
		if err != nil {
			return nil, err
		}
		loc := re.FindIndex(b)
		if loc == nil {
			return nil, fmt.Errorf("could not match %q", s)
		}
		return loc, nil
	}

	loc, err := match(start)
	if err != nil {
		return nil, err
	}
	b = b[loc[0]:]

	loc, err = match(end)
	if err != nil {
		return nil, err
	}
	b = b[:loc[1]]

	return b, nil
}
