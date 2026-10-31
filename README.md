# nocomments

`nocomments` fails when Go, shell or TOML code in a repository contains comments.

A comment goes stale silently: neither the compiler nor the tests check it, and it drifts away from the code and from the docs that say the same thing. Knowledge that should live belongs where it is checked (a name, a test) or where people look for it (an ADR, a README). `nocomments` keeps comments from creeping back, so you learn about one from your check script or pre-commit hook, not in review.

## Install

```bash
go install github.com/MikhailEpatko/nocomments@latest
```

Requires Go 1.22 or newer. No third-party dependencies.

## Usage

Run it from the repository root:

```bash
nocomments
```

It walks the current directory and prints every comment it finds as `<path>:<line>: <text>`, followed by a hint line:

```text
main.go:12: // parseArgs parses the arguments
scripts/check.sh:2: # run all checks
comments found: move the knowledge into a name, a test or the docs, then delete the comment
```

- Paths are relative to the current directory and use `/` on every OS.
- Findings come in tree walk order (directory entries by name), by line within a file.
- A multi-line `/* … */` is reported by its first line.
- If a file or directory cannot be read, the error is printed and the walk goes on.

| Result | Output | Exit code |
| --- | --- | --- |
| No comments | nothing | 0 |
| Comments found | findings, then the hint | 1 |
| A file could not be read | the error (and the hint if there are findings) | 1 |

`nocomments` does not use git, so it also works on a checked-out copy of the index without `.git`, which is how a pre-commit hook can check exactly what is being committed.

## What is checked

| Files | A comment is |
| --- | --- |
| `*.go`, tests included | any `//` or `/* */` outside string literals, found with `go/scanner` |
| `*.sh` and every file directly in the top-level `.githooks/` | a line whose first non-blank character is `#` |
| `*.toml` | a line whose first non-blank character is `#` |

Only the `.git` and `.claude` directories are skipped, so `testdata/` and `vendor/` are checked like any other directory. Files of other kinds (Markdown, YAML, JSON, `go.mod`, files without an extension outside `.githooks/`) are not code and are not read.

## Allowed directives

Tool directives are not comments, as long as they carry no explanation:

- `//go:…` and `//nolint…` with no ` //` after them: `//go:build linux` passes, `//nolint:errcheck // reason` is a finding;
- `// gitleaks:allow` with nothing else in the comment (spacing does not matter): `// gitleaks:allow, it is an example` is a finding;
- a shebang on the first line of a shell script: `#!` on any other line is a finding.

## Limitations

- A trailing shell comment (`cmd # text`) is not detected: without a shell parser it cannot be told apart from `$#` or `${#x}`. Catch it in review.
- Any text right after `//go:` (`//go:this is a comment`) is taken for a directive.
- There is no way to exclude a directory: vendored Go code in `vendor/` is reported like your own.
- Heredocs and multi-line TOML strings are not parsed, so a data line in them that starts with `#` is a finding. Print a shebang for a generated script with a separate command: `{ echo '#!/bin/sh'; cat <<'EOF'; } >file`.

## Development

```bash
gofmt -l .
go vet ./...
go test -race ./...
go run .
```

The tests call `run(fs.FS, io.Writer)` with an in-memory `fstest.MapFS` and check only stdout and the exit code. The last command checks this repository itself.

## License

[MIT](LICENSE)
