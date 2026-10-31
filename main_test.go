package main

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

const wantHint = "comments found: move the knowledge into a name, a test or the docs, then delete the comment\n"

type checkCase struct {
	name     string
	files    fs.FS
	wantCode int
	want     string
}

func TestRunReportsGoComments(t *testing.T) {
	assertChecks(t, []checkCase{
		{
			name:     "line comment",
			files:    fstest.MapFS{"a.go": file("package a\n\n// note\nvar x = 1\n")},
			wantCode: 1,
			want:     "a.go:3: // note\n" + wantHint,
		},
		{
			name:     "trailing comment",
			files:    fstest.MapFS{"a.go": file("package a\n\nvar x = 1 // note\n")},
			wantCode: 1,
			want:     "a.go:3: // note\n" + wantHint,
		},
		{
			name:     "single-line block comment",
			files:    fstest.MapFS{"a.go": file("package a\n\nvar x = /* note */ 1\n")},
			wantCode: 1,
			want:     "a.go:3: /* note */\n" + wantHint,
		},
		{
			name:     "multi-line block comment is reported by its first line",
			files:    fstest.MapFS{"a.go": file("package a\n\n/* first\nsecond */\nvar x = 1\n")},
			wantCode: 1,
			want:     "a.go:3: /* first\n" + wantHint,
		},
		{
			name:     "comment in a test file",
			files:    fstest.MapFS{"a_test.go": file("package a\n\n// note\n")},
			wantCode: 1,
			want:     "a_test.go:3: // note\n" + wantHint,
		},
		{
			name:  "go and nolint directives are not comments",
			files: fstest.MapFS{"a.go": file("//go:build linux\n\npackage a\n\nvar x = f() //nolint:errcheck\n")},
		},
		{
			name:     "nolint with an explanation is a comment",
			files:    fstest.MapFS{"a.go": file("package a\n\nvar x = f() //nolint:errcheck // reason\n")},
			wantCode: 1,
			want:     "a.go:3: //nolint:errcheck // reason\n" + wantHint,
		},
		{
			name:     "go:build with an explanation is a comment",
			files:    fstest.MapFS{"a.go": file("//go:build linux // linux only\n\npackage a\n")},
			wantCode: 1,
			want:     "a.go:1: //go:build linux // linux only\n" + wantHint,
		},
		{
			name:  "gitleaks:allow is not a comment",
			files: fstest.MapFS{"a.go": file("package a\n\nvar x = \"y\" // gitleaks:allow\n")},
		},
		{
			name:     "gitleaks:allow with an explanation is a comment",
			files:    fstest.MapFS{"a.go": file("package a\n\nvar x = \"y\" // gitleaks:allow, just an example\n")},
			wantCode: 1,
			want:     "a.go:3: // gitleaks:allow, just an example\n" + wantHint,
		},
		{
			name:  "// and /* inside string literals are not comments",
			files: fstest.MapFS{"a.go": file("package a\n\nvar x = \"https://example.com /* y */\"\nvar y = `{\"url\": \"https://example.com\"} /* z`\n")},
		},
	})
}

func TestRunReportsShellComments(t *testing.T) {
	assertChecks(t, []checkCase{
		{
			name:     "line starting with #",
			files:    fstest.MapFS{"s.sh": file("set -eu\n# note\necho ok\n")},
			wantCode: 1,
			want:     "s.sh:2: # note\n" + wantHint,
		},
		{
			name:     "line starting with # after indentation",
			files:    fstest.MapFS{"s.sh": file("f() {\n\t  # note\n\techo ok\n}\n")},
			wantCode: 1,
			want:     "s.sh:2: # note\n" + wantHint,
		},
		{
			name:  "shebang on the first line is not a comment",
			files: fstest.MapFS{"s.sh": file("#!/bin/sh\nset -eu\n")},
		},
		{
			name:     "#! after the first line is a comment",
			files:    fstest.MapFS{"s.sh": file("set -eu\n#!/bin/sh\n")},
			wantCode: 1,
			want:     "s.sh:2: #!/bin/sh\n" + wantHint,
		},
		{
			name:  "# inside a line is not a comment",
			files: fstest.MapFS{"s.sh": file("echo $# ${#x} \"#\"\n")},
		},
		{
			name:  "trailing # is not detected",
			files: fstest.MapFS{"s.sh": file("echo ok # note\n")},
		},
	})
}

func TestRunSelectsFiles(t *testing.T) {
	assertChecks(t, []checkCase{
		{
			name:     "hook without extension is checked as shell",
			files:    fstest.MapFS{".githooks/pre-commit": file("#!/bin/sh\n# note\n")},
			wantCode: 1,
			want:     ".githooks/pre-commit:2: # note\n" + wantHint,
		},
		{
			name:     "TOML is checked and a shebang in it is no exception",
			files:    fstest.MapFS{"a.toml": file("#!/x\ntitle = \"a#b\"\n  # note\n")},
			wantCode: 1,
			want:     "a.toml:1: #!/x\na.toml:3: # note\n" + wantHint,
		},
		{
			name: ".claude and .git directories are skipped",
			files: fstest.MapFS{
				".claude/skills/wizard/template.sh": file("# note\n"),
				".claude/a.go":                      file("package a // note\n"),
				".git/hooks/pre-commit.sample":      file("# note\n"),
				".git/a.sh":                         file("# note\n"),
			},
		},
		{
			name: "Markdown, go.mod, YAML and files without extension are not code",
			files: fstest.MapFS{
				"README.md":        file("# Title\n"),
				"go.mod":           file("// note\nmodule a\n"),
				"a.yml":            file("# note\n"),
				"testdata/a.json":  file("// note\n"),
				"testdata/fixture": file("# note\n"),
			},
		},
	})
}

func TestRunOutput(t *testing.T) {
	assertChecks(t, []checkCase{
		{
			name: "no comments, no output",
			files: fstest.MapFS{
				"main.go":          file("package main\n\nfunc main() {}\n"),
				"scripts/check.sh": file("#!/bin/sh\nset -eu\n"),
			},
		},
		{
			name: "findings in tree walk order, by line within a file, one hint at the end",
			files: fstest.MapFS{
				"scripts/b.sh":         file("# b1\necho\n# b2\n"),
				"internal/app/run.go":  file("package app\n\n// r\n"),
				".githooks/pre-commit": file("# h\n"),
				"main.go":              file("// m1\npackage main // m2\n"),
				"a.toml":               file("# t\n"),
				"a/b.sh":               file("# ab\n"),
			},
			wantCode: 1,
			want: ".githooks/pre-commit:1: # h\n" +
				"a/b.sh:1: # ab\n" +
				"a.toml:1: # t\n" +
				"internal/app/run.go:3: // r\n" +
				"main.go:1: // m1\n" +
				"main.go:2: // m2\n" +
				"scripts/b.sh:1: # b1\n" +
				"scripts/b.sh:3: # b2\n" +
				wantHint,
		},
	})
}

func TestRunReportsUnreadableFile(t *testing.T) {
	assertChecks(t, []checkCase{
		{
			name: "error without findings is printed without the hint",
			files: unreadable{
				files: fstest.MapFS{
					"scripts/a.sh":      file("#!/bin/sh\n"),
					"scripts/broken.sh": file("#!/bin/sh\n"),
				},
				name: "scripts/broken.sh",
			},
			wantCode: 1,
			want:     "open scripts/broken.sh: permission denied\n",
		},
		{
			name: "files after the error are still checked",
			files: unreadable{
				files: fstest.MapFS{
					"scripts/broken.sh": file("#!/bin/sh\n"),
					"scripts/c.sh":      file("# note\n"),
				},
				name: "scripts/broken.sh",
			},
			wantCode: 1,
			want:     "open scripts/broken.sh: permission denied\nscripts/c.sh:1: # note\n" + wantHint,
		},
	})
}

type unreadable struct {
	files fstest.MapFS
	name  string
}

func (u unreadable) Open(name string) (fs.File, error) {
	if name == u.name {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	return u.files.Open(name)
}

func file(data string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(data)}
}

func assertChecks(t *testing.T, tests []checkCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout strings.Builder
			code := run(tt.files, &stdout)
			if code != tt.wantCode {
				t.Errorf("exit code %d, want %d", code, tt.wantCode)
			}
			if stdout.String() != tt.want {
				t.Errorf("stdout:\n%s\nwant:\n%s", stdout.String(), tt.want)
			}
		})
	}
}
