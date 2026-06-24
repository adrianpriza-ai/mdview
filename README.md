# mdview

[![Build](https://github.com/adrianpriza-ai/mdview/actions/workflows/build.yml/badge.svg)](https://github.com/adrianpriza-ai/mdview/actions/workflows/build.yml)

Render Markdown files in your terminal with ANSI color and formatting.
Zero external dependencies — pure Go stdlib.

## Build

```bash
cd mdview
go build -o mdview .
```

Or install to `$GOPATH/bin`:

```bash
go install .
```

## Usage

```
mdview [--width N] <file.md>
cat README.md | mdview
```

`mdview` reads from a file path or from stdin when no path is given.

Terminal width is detected automatically via `TIOCGWINSZ` (syscall) with a
fallback of 80 columns. Use `--width` to override.

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--width N` | `-w N` | Wrap output to N columns instead of auto-detected terminal width |

### Examples

```bash
# View a local file
mdview README.md

# Specify a fixed render width
mdview --width 100 README.md
mdview -w 100 README.md

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

| Feature | Syntax |
|---|---|
| Headings (ATX) | `#` through `######` |
| Headings (setext) | underline with `===` / `---` |
| Bold | `**text**` or `__text__` |
| Italic | `*text*` or `_text_` |
| Bold + italic | `***text***` |
| Strikethrough | `~~text~~` |
| Inline code | `` `code` `` |
| Fenced code blocks | ` ``` ` with optional language label |
| Blockquotes | `> text` |
| Unordered lists | `- item` / `* item` |
| Ordered lists | `1. item` |
| Nested lists | two-space indent |
| Tables (GFM) | pipe-delimited, column alignment |
| Links | `[text](url)` |
| Auto-links | `<https://example.com>` |
| Images | `![alt](url)` → renders as `[alt]` label |
| Horizontal rules | `---` / `***` / `___` |
| Centered blocks | `<div align="center">` / `<p align="center">` |

## Table rendering

Tables automatically shrink to fit the terminal (or `--width`). Cells
word-wrap across multiple rows when needed; words wider than their column are
hard-broken. Inline markup — bold, italic, inline code, links — is fully
rendered inside table cells.

## Centered blocks

HTML center blocks render all content flush-centered to the terminal width.
Inside a `<div align="center">` or `<p align="center">` block, the following
are supported and styled:

- ATX markdown headings (`# h1` through `###### h6`) — bold + color by level
- HTML headings (`<h1>` through `<h6>`)
- Inline images (`<img>`) → rendered as `[alt]`
- Inline links (`<a>`) → rendered as link text
- HTML line breaks (`<br>`)
- HTML entities (`&amp;`, `&lt;`, `&gt;`, `&quot;`, `&#NNN;`, `&shy;`, …)
