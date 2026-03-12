# mdserve

Simple CLI that serves markdown files rendered as HTML.

## Install

```sh
go install github.com/gonzaloserrano/mdserve@latest
```

Or download a binary from [releases](https://github.com/gonzaloserrano/mdserve/releases).

## Usage

```sh
mdserve [path]        # default: current directory
mdserve -p 3000 .     # custom port (default: 8080)
mdserve README.md     # serve a single file
mdserve docs/         # serve a directory tree
```

- **Directory mode**: file browser listing `.md` files and subdirectories
- **File mode**: renders a single markdown file as HTML
- Non-markdown files and missing paths return 404

## Features

- Tables, strikethrough, and autolinks via [goldmark](https://github.com/yuin/goldmark)
- [Mermaid](https://mermaid.js.org/) diagram rendering (JS embedded in binary, works offline)
- Breadcrumb navigation in subdirectories
- Minimal inline CSS, no external dependencies
- Directory traversal prevention via `os.OpenRoot`
