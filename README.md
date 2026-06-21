# mdview

[![Build](https://github.com/adrianpriza-ai/mdview/actions/workflows/build.yml/badge.svg)](https://github.com/adrianpriza-ai/mdview/actions/workflows/build.yml)

Render Markdown files in your terminal with ANSI color and formatting.
Zero external dependencies — pure Go stdlib.

## Build

```bash
cd tools/mdview
go build -o mdview .
```

Or install to `$GOPATH/bin`:

```bash
go install .
```

## Usage

```
mdview <file.md>
cat README.md | mdview
```

### Examples

```bash
# View a local file
mdview README.md

# Pipe from stdin
cat CHANGELOG.md | mdview

# Pipe to a pager (preserves colors)
mdview README.md | less -R

# View a remote file
curl -s https://raw.githubusercontent.com/cli/cli/trunk/README.md | mdview
```

## Simple Preview

[![asciicast](https://asciinema.org/a/VOJaCl5d3FsVUXup.svg)](https://asciinema.org/a/VOJaCl5d3FsVUXup)

## Supported Markdown

| Feature         | Syntax                        |
|-----------------|-------------------------------|
| Headings        | `#` through `######`          |
| Bold            | `**text**` or `__text__`      |
| Italic          | `*text*` or `_text_`          |
| Bold + Italic   | `***text***`                  |
| Strikethrough   | `~~text~~`                    |
| Inline code     | `` `code` ``                  |
| Fenced code     | ` ``` ` with optional lang    |
| Blockquotes     | `> text`                      |
| Unordered lists | `- item` / `* item`           |
| Ordered lists   | `1. item`                     |
| Nested lists    | two-space indent               |
| Tables          | GFM pipe tables               |
| Links           | `[text](url)`                 |
| Images          | `![alt](url)` → label         |
| Horizontal rule | `---` / `***`                 |
| Setext headings | underline with `===` / `---`  |
