package main

import (
	"bytes"
	"fmt"
	"go/scanner"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

const hint = "comments found: move the knowledge into a name, a test or the docs, then delete the comment"

type comment struct {
	line int
	text string
}

func main() {
	os.Exit(run(os.DirFS("."), os.Stdout))
}

func run(fsys fs.FS, stdout io.Writer) int {
	found, failed := false, false
	reportError := func(err error) error {
		fmt.Fprintln(stdout, err)
		failed = true
		return nil
	}
	_ = fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return reportError(err)
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".claude" {
				return fs.SkipDir
			}
			return nil
		}
		find := commentFinder(name)
		if find == nil {
			return nil
		}
		src, err := fs.ReadFile(fsys, name)
		if err != nil {
			return reportError(err)
		}
		for _, c := range find(src) {
			fmt.Fprintf(stdout, "%s:%d: %s\n", name, c.line, c.text)
			found = true
		}
		return nil
	})
	if found {
		fmt.Fprintln(stdout, hint)
	}
	if found || failed {
		return 1
	}
	return 0
}

func commentFinder(name string) func(src []byte) []comment {
	ext := path.Ext(name)
	switch {
	case ext == ".go":
		return goComments
	case ext == ".sh" || path.Dir(name) == ".githooks":
		return shellComments
	case ext == ".toml":
		return hashComments
	}
	return nil
}

func goComments(src []byte) []comment {
	fset := token.NewFileSet()
	file := fset.AddFile("", -1, len(src))
	var s scanner.Scanner
	s.Init(file, src, nil, scanner.ScanComments)
	var comments []comment
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			return comments
		}
		if tok == token.COMMENT && !isDirective(lit) {
			firstLine, _, _ := strings.Cut(lit, "\n")
			comments = append(comments, comment{file.Line(pos), firstLine})
		}
	}
}

func isDirective(text string) bool {
	toolDirective := strings.HasPrefix(text, "//go:") || strings.HasPrefix(text, "//nolint")
	explained := strings.Contains(text, " //")
	return toolDirective && !explained ||
		strings.TrimSpace(strings.TrimPrefix(text, "//")) == "gitleaks:allow"
}

func shellComments(src []byte) []comment {
	comments := hashComments(src)
	if bytes.HasPrefix(src, []byte("#!")) {
		return comments[1:]
	}
	return comments
}

func hashComments(src []byte) []comment {
	var comments []comment
	for i, line := range strings.Split(string(src), "\n") {
		text := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(text, "#") {
			comments = append(comments, comment{i + 1, text})
		}
	}
	return comments
}
