package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf8"
	"unsafe"
)

// ANSI escape codes
const (
	reset     = "\033[0m"
	bold      = "\033[1m"
	italic    = "\033[3m"
	dim       = "\033[2m"
	underline = "\033[4m"
	strike    = "\033[9m"

	fgBlack   = "\033[30m"
	fgRed     = "\033[31m"
	fgGreen   = "\033[32m"
	fgYellow  = "\033[33m"
	fgBlue    = "\033[34m"
	fgMagenta = "\033[35m"
	fgCyan    = "\033[36m"
	fgWhite   = "\033[37m"

	fgBrightBlack   = "\033[90m"
	fgBrightRed     = "\033[91m"
	fgBrightGreen   = "\033[92m"
	fgBrightYellow  = "\033[93m"
	fgBrightBlue    = "\033[94m"
	fgBrightMagenta = "\033[95m"
	fgBrightCyan    = "\033[96m"
	fgBrightWhite   = "\033[97m"

	bgBlack = "\033[40m"
	bgWhite = "\033[47m"

	bgBrightBlack = "\033[100m"
)

// winsize mirrors the kernel struct used by TIOCGWINSZ.
type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

// termWidth queries the terminal dimensions via ioctl.
// It tries stdout first, then stderr, then stdin.
// Falls back to 80 columns when none of them is a tty (e.g. when piping).
func termWidth() int {
	for _, fd := range []uintptr{
		uintptr(syscall.Stdout),
		uintptr(syscall.Stderr),
		uintptr(syscall.Stdin),
	} {
		ws := &winsize{}
		rc, _, _ := syscall.Syscall(
			syscall.SYS_IOCTL,
			fd,
			syscall.TIOCGWINSZ,
			uintptr(unsafe.Pointer(ws)),
		)
		if rc == 0 && ws.Col > 0 {
			return int(ws.Col)
		}
	}
	return 80 // fallback when not attached to a tty
}

// repeat returns a string of n copies of s.
func repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(s, n)
}

// ruler returns a horizontal rule of width w.
func ruler(char string, w int) string {
	return fgBrightBlack + repeat(char, w) + reset
}

// stripAnsi removes ANSI escape sequences to measure visible length.
var ansiEscapeRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func visibleLen(s string) int {
	return utf8.RuneCountInString(ansiEscapeRe.ReplaceAllString(s, ""))
}

// padRight pads s with spaces to reach visible length n.
func padRight(s string, n int) string {
	l := visibleLen(s)
	if l >= n {
		return s
	}
	return s + repeat(" ", n-l)
}

// center centers s in a field of width w.
func center(s string, w int) string {
	l := visibleLen(s)
	if l >= w {
		return s
	}
	pad := (w - l) / 2
	return repeat(" ", pad) + s + repeat(" ", w-l-pad)
}

// Inline renderer

var (
	reBoldItalic = regexp.MustCompile(`\*{3}(.+?)\*{3}|_{3}(.+?)_{3}`)
	reBold       = regexp.MustCompile(`\*{2}(.+?)\*{2}|_{2}(.+?)_{2}`)
	reItalic     = regexp.MustCompile(`\*([^*]+?)\*|_([^_]+?)_`)
	reStrike     = regexp.MustCompile(`~~(.+?)~~`)
	reInlineCode = regexp.MustCompile("`([^`]+)`")
	reLink       = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reImage      = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	reAutolink   = regexp.MustCompile(`<(https?://[^>]+)>`)
	reBareURL    = regexp.MustCompile(`(?:^|[\s(])(https?://\S+)`)
)

// renderInline processes inline markdown within a single line.
func renderInline(s string) string {
	// images → label
	s = reImage.ReplaceAllStringFunc(s, func(m string) string {
		sub := reImage.FindStringSubmatch(m)
		alt := sub[1]
		if alt == "" {
			alt = "image"
		}
		return fgBrightBlack + "[" + alt + "]" + reset
	})

	// links → text (url)
	s = reLink.ReplaceAllStringFunc(s, func(m string) string {
		sub := reLink.FindStringSubmatch(m)
		return bold + fgBrightBlue + sub[1] + reset + fgBrightBlack + " (" + sub[2] + ")" + reset
	})

	// autolinks
	s = reAutolink.ReplaceAllStringFunc(s, func(m string) string {
		sub := reAutolink.FindStringSubmatch(m)
		return fgBrightBlue + underline + sub[1] + reset
	})

	// inline code (do this before bold/italic to avoid re-processing)
	s = reInlineCode.ReplaceAllStringFunc(s, func(m string) string {
		sub := reInlineCode.FindStringSubmatch(m)
		return bgBrightBlack + fgBrightWhite + " " + sub[1] + " " + reset
	})

	// bold+italic
	s = reBoldItalic.ReplaceAllStringFunc(s, func(m string) string {
		sub := reBoldItalic.FindStringSubmatch(m)
		inner := sub[1]
		if inner == "" {
			inner = sub[2]
		}
		return bold + italic + inner + reset
	})

	// bold
	s = reBold.ReplaceAllStringFunc(s, func(m string) string {
		sub := reBold.FindStringSubmatch(m)
		inner := sub[1]
		if inner == "" {
			inner = sub[2]
		}
		return bold + inner + reset
	})

	// italic
	s = reItalic.ReplaceAllStringFunc(s, func(m string) string {
		sub := reItalic.FindStringSubmatch(m)
		inner := sub[1]
		if inner == "" {
			inner = sub[2]
		}
		return italic + inner + reset
	})

	// strikethrough
	s = reStrike.ReplaceAllStringFunc(s, func(m string) string {
		sub := reStrike.FindStringSubmatch(m)
		return strike + fgBrightBlack + sub[1] + reset
	})

	return s
}

// Block-level parser

type lineKind int

const (
	kindBlank lineKind = iota
	kindHeading
	kindHR
	kindFence
	kindBlockquote
	kindListItem
	kindIndentedCode
	kindTable
	kindHTMLComment
	kindParagraph
)

type parsedLine struct {
	raw   string
	kind  lineKind
	level int  // heading level 1-6
	depth int  // list nesting
	ord   bool // ordered list
	index int  // ordered list number
}

func classifyLine(s string) parsedLine {
	p := parsedLine{raw: s}

	// Blank
	if strings.TrimSpace(s) == "" {
		p.kind = kindBlank
		return p
	}

	// HTML comment
	if strings.HasPrefix(strings.TrimSpace(s), "<!--") {
		p.kind = kindHTMLComment
		return p
	}

	// Fenced code block (``` or ~~~)
	trimmed := strings.TrimLeft(s, " \t")
	if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
		p.kind = kindFence
		return p
	}

	// ATX heading
	if strings.HasPrefix(trimmed, "#") {
		level := 0
		for _, ch := range trimmed {
			if ch == '#' {
				level++
			} else {
				break
			}
		}
		if level <= 6 && len(trimmed) > level && trimmed[level] == ' ' {
			p.kind = kindHeading
			p.level = level
			p.raw = strings.TrimSpace(trimmed[level+1:])
			return p
		}
	}

	// Setext heading (=== or ---)
	allEq := regexp.MustCompile(`^=+\s*$`)
	allDash := regexp.MustCompile(`^-+\s*$`)
	if allEq.MatchString(trimmed) {
		p.kind = kindHeading
		p.level = 1
		return p
	}
	if allDash.MatchString(trimmed) && len(trimmed) >= 3 {
		p.kind = kindHR
		return p
	}

	// Thematic break (--- *** ___)
	hrRe := regexp.MustCompile(`^(\*{3,}|-{3,}|_{3,})\s*$`)
	if hrRe.MatchString(trimmed) {
		p.kind = kindHR
		return p
	}

	// Blockquote
	if strings.HasPrefix(trimmed, "> ") || trimmed == ">" {
		p.kind = kindBlockquote
		if len(trimmed) > 2 {
			p.raw = trimmed[2:]
		} else {
			p.raw = ""
		}
		return p
	}

	// Ordered list
	ordRe := regexp.MustCompile(`^(\s*)(\d+)\.\s+(.*)`)
	if m := ordRe.FindStringSubmatch(s); m != nil {
		p.kind = kindListItem
		p.ord = true
		p.depth = len(m[1]) / 2
		fmt.Sscanf(m[2], "%d", &p.index)
		p.raw = m[3]
		return p
	}

	// Unordered list
	ulRe := regexp.MustCompile(`^(\s*)[-*+]\s+(.*)`)
	if m := ulRe.FindStringSubmatch(s); m != nil {
		p.kind = kindListItem
		p.depth = len(m[1]) / 2
		p.raw = m[2]
		return p
	}

	// Indented code (4 spaces or tab)
	if strings.HasPrefix(s, "    ") || strings.HasPrefix(s, "\t") {
		p.kind = kindIndentedCode
		if strings.HasPrefix(s, "\t") {
			p.raw = s[1:]
		} else {
			p.raw = s[4:]
		}
		return p
	}

	// Table row (contains |)
	if strings.Contains(trimmed, "|") {
		p.kind = kindTable
		return p
	}

	p.kind = kindParagraph
	return p
}

// Renderer

type renderer struct {
	w     int
	lines []string
	out   *strings.Builder
}

func newRenderer() *renderer {
	return &renderer{
		w:   termWidth(),
		out: &strings.Builder{},
	}
}

func newRendererWidth(w int) *renderer {
	return &renderer{
		w:   w,
		out: &strings.Builder{},
	}
}

func (r *renderer) emit(s string) {
	r.out.WriteString(s)
	r.out.WriteByte('\n')
}

func (r *renderer) blank() {
	r.out.WriteByte('\n')
}

func (r *renderer) renderHeading(level int, text string) {
	rendered := renderInline(text)
	switch level {
	case 1:
		r.blank()
		inner := bold + fgBrightWhite + " " + strings.ToUpper(rendered) + " " + reset
		line := fgBrightMagenta + repeat("█", 2) + reset + " " + inner
		r.emit(line)
		r.emit(fgBrightMagenta + repeat("─", r.w) + reset)
	case 2:
		r.blank()
		line := bold + fgBrightCyan + "## " + rendered + reset
		r.emit(line)
		r.emit(fgCyan + repeat("─", r.w/2) + reset)
	case 3:
		r.blank()
		r.emit(bold + fgBrightYellow + "### " + rendered + reset)
	case 4:
		r.blank()
		r.emit(bold + fgYellow + "#### " + rendered + reset)
	case 5:
		r.blank()
		r.emit(bold + fgBrightBlack + "##### " + rendered + reset)
	default:
		r.blank()
		r.emit(bold + fgBrightBlack + "###### " + rendered + reset)
	}
}

func (r *renderer) renderHR() {
	r.blank()
	r.emit(fgBrightBlack + repeat("─", r.w) + reset)
	r.blank()
}

func (r *renderer) renderCodeBlock(lang string, lines []string) {
	r.blank()
	header := bgBrightBlack + fgBrightWhite + bold
	if lang != "" {
		header += "  " + strings.ToUpper(lang) + "  "
	} else {
		header += "  CODE  "
	}
	header += reset
	r.emit(header)
	bar := fgBrightBlack + "│" + reset
	for _, l := range lines {
		r.emit(bar + " " + fgBrightGreen + l + reset)
	}
	r.emit(fgBrightBlack + repeat("─", r.w) + reset)
	r.blank()
}

func (r *renderer) renderBlockquote(lines []string) {
	r.blank()
	for _, l := range lines {
		r.emit(fgBrightBlack + "▌ " + reset + italic + fgBrightWhite + renderInline(l) + reset)
	}
	r.blank()
}

func (r *renderer) renderListItem(depth int, ord bool, index int, text string) {
	indent := repeat("  ", depth)
	bullet := fgBrightMagenta + "•" + reset
	if ord {
		bullet = fgBrightMagenta + fmt.Sprintf("%d.", index) + reset
	}
	if depth > 0 {
		bullet = fgBrightBlack + "◦" + reset
		if ord {
			bullet = fgBrightBlack + fmt.Sprintf("%d.", index) + reset
		}
	}
	r.emit(indent + bullet + " " + renderInline(text))
}

func (r *renderer) renderTable(rows []string) {
	if len(rows) < 2 {
		for _, row := range rows {
			r.emit(renderInline(row))
		}
		return
	}

	// Parse cells
	parseCells := func(row string) []string {
		row = strings.Trim(row, "| \t")
		parts := strings.Split(row, "|")
		cells := make([]string, len(parts))
		for i, p := range parts {
			cells[i] = strings.TrimSpace(p)
		}
		return cells
	}

	// Check if row 1 is a separator
	sepRe := regexp.MustCompile(`^[\s|:\-]+$`)
	hasSep := len(rows) >= 2 && sepRe.MatchString(rows[1])

	allRows := [][]string{}
	for i, row := range rows {
		if hasSep && i == 1 {
			continue
		}
		allRows = append(allRows, parseCells(row))
	}

	if len(allRows) == 0 {
		return
	}

	// Compute column widths
	numCols := 0
	for _, row := range allRows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}
	colWidths := make([]int, numCols)
	for _, row := range allRows {
		for i, cell := range row {
			if i < numCols && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	hline := func(left, mid, right, fill string) string {
		parts := make([]string, numCols)
		for i, w := range colWidths {
			parts[i] = repeat(fill, w+2)
		}
		return fgBrightBlack + left + strings.Join(parts, mid) + right + reset
	}

	r.blank()
	r.emit(hline("┌", "┬", "┐", "─"))
	for ri, row := range allRows {
		line := fgBrightBlack + "│" + reset
		for ci := 0; ci < numCols; ci++ {
			cell := ""
			if ci < len(row) {
				cell = row[ci]
			}
			if ri == 0 {
				cell = bold + fgBrightWhite + cell + reset
			} else {
				cell = renderInline(cell)
			}
			line += " " + padRight(cell, colWidths[ci]) + " " + fgBrightBlack + "│" + reset
		}
		r.emit(line)
		if ri == 0 {
			r.emit(hline("├", "┼", "┤", "─"))
		}
	}
	r.emit(hline("└", "┴", "┘", "─"))
	r.blank()
}

func (r *renderer) renderParagraph(lines []string) {
	text := strings.Join(lines, " ")
	text = renderInline(text)
	// simple word-wrap
	words := strings.Fields(text)
	line := ""
	for _, w := range words {
		if visibleLen(line)+visibleLen(w)+1 > r.w && line != "" {
			r.emit(line)
			line = w
		} else {
			if line == "" {
				line = w
			} else {
				line += " " + w
			}
		}
	}
	if line != "" {
		r.emit(line)
	}
}

// Main parse loop

func render(src string, w int) string {
	r := newRendererWidth(w)
	scanner := bufio.NewScanner(strings.NewReader(src))
	rawLines := []string{}
	for scanner.Scan() {
		rawLines = append(rawLines, scanner.Text())
	}

	inFence := false
	fenceLang := ""
	fenceLines := []string{}

	inBlockquote := false
	bqLines := []string{}

	paraLines := []string{}

	inTable := false
	tableRows := []string{}

	flushParagraph := func() {
		if len(paraLines) > 0 {
			r.renderParagraph(paraLines)
			r.blank()
			paraLines = nil
		}
	}

	flushBlockquote := func() {
		if len(bqLines) > 0 {
			r.renderBlockquote(bqLines)
			bqLines = nil
		}
		inBlockquote = false
	}

	flushTable := func() {
		if len(tableRows) > 0 {
			r.renderTable(tableRows)
			tableRows = nil
		}
		inTable = false
	}

	prevSetextCandidate := ""

	for i, raw := range rawLines {
		// Handle setext-style headings: check next line
		if !inFence && i+1 < len(rawLines) {
			next := strings.TrimSpace(rawLines[i+1])
			allEq := regexp.MustCompile(`^=+$`)
			allDash2 := regexp.MustCompile(`^-{2,}$`)
			if allEq.MatchString(next) && strings.TrimSpace(raw) != "" {
				prevSetextCandidate = strings.TrimSpace(raw)
				_ = prevSetextCandidate
				flushParagraph()
				flushBlockquote()
				flushTable()
				r.renderHeading(1, strings.TrimSpace(raw))
				continue
			}
			if allDash2.MatchString(next) && strings.TrimSpace(raw) != "" && !strings.HasPrefix(strings.TrimSpace(raw), "#") {
				flushParagraph()
				flushBlockquote()
				flushTable()
				r.renderHeading(2, strings.TrimSpace(raw))
				continue
			}
		}
		// Skip the setext underline line itself
		allEqRe := regexp.MustCompile(`^=+\s*$`)
		allDashRe := regexp.MustCompile(`^-{2,}\s*$`)
		if !inFence && i > 0 {
			prev := strings.TrimSpace(rawLines[i-1])
			cur := strings.TrimSpace(raw)
			if (allEqRe.MatchString(cur) || allDashRe.MatchString(cur)) && prev != "" && !strings.HasPrefix(prev, "#") {
				continue
			}
		}

		// Inside a fenced code block
		if inFence {
			trimmed := strings.TrimLeft(raw, " \t")
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				r.renderCodeBlock(fenceLang, fenceLines)
				fenceLines = nil
				fenceLang = ""
				inFence = false
			} else {
				fenceLines = append(fenceLines, raw)
			}
			continue
		}

		p := classifyLine(raw)

		switch p.kind {
		case kindFence:
			flushParagraph()
			flushBlockquote()
			flushTable()
			inFence = true
			trimmed := strings.TrimLeft(raw, " \t~`")
			fenceLang = strings.TrimSpace(trimmed)

		case kindBlank:
			flushParagraph()
			flushBlockquote()
			flushTable()
			r.blank()

		case kindHTMLComment:
			// skip

		case kindHeading:
			flushParagraph()
			flushBlockquote()
			flushTable()
			r.renderHeading(p.level, p.raw)

		case kindHR:
			flushParagraph()
			flushBlockquote()
			flushTable()
			r.renderHR()

		case kindBlockquote:
			flushParagraph()
			flushTable()
			inBlockquote = true
			bqLines = append(bqLines, p.raw)

		case kindListItem:
			flushParagraph()
			flushBlockquote()
			flushTable()
			r.renderListItem(p.depth, p.ord, p.index, p.raw)

		case kindTable:
			flushParagraph()
			flushBlockquote()
			inTable = true
			tableRows = append(tableRows, raw)

		case kindIndentedCode:
			flushParagraph()
			flushBlockquote()
			flushTable()
			r.renderCodeBlock("", []string{p.raw})

		case kindParagraph:
			if inBlockquote {
				flushBlockquote()
			}
			if inTable {
				flushTable()
			}
			paraLines = append(paraLines, strings.TrimSpace(raw))
		}
	}

	// Flush any remaining blocks
	flushParagraph()
	flushBlockquote()
	flushTable()
	if inFence && len(fenceLines) > 0 {
		r.renderCodeBlock(fenceLang, fenceLines)
	}

	return r.out.String()
}

// Entry point

func usage() {
	fmt.Fprintf(os.Stderr, `%smdview%s — render Markdown in your terminal

%sUSAGE%s
  mdview [--width N] <file.md>
  cat README.md | mdview [--width N]

%sFLAGS%s
  -w, --width N   Override render width (default: auto-detect from terminal)
  -h, --help      Show this help message

%sEXAMPLES%s
  mdview README.md
  mdview --width 120 README.md
  mdview CHANGELOG.md | less -R
  curl -s https://raw.githubusercontent.com/cli/cli/trunk/README.md | mdview

`,
		bold+fgBrightMagenta, reset,
		bold+fgBrightWhite, reset,
		bold+fgBrightWhite, reset,
		bold+fgBrightWhite, reset,
	)
}

// parseArgs splits os.Args[1:] into (width, file-or-empty, error).
// width==0 means "auto-detect".
func parseArgs() (width int, file string, err error) {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-h" || a == "--help":
			usage()
			os.Exit(0)
		case a == "-w" || a == "--width":
			if i+1 >= len(args) {
				return 0, "", fmt.Errorf("flag %q requires a numeric argument", a)
			}
			i++
			n, nerr := strconv.Atoi(args[i])
			if nerr != nil || n < 20 {
				return 0, "", fmt.Errorf("--width must be an integer >= 20, got %q", args[i])
			}
			width = n
		case strings.HasPrefix(a, "--width="):
			val := strings.TrimPrefix(a, "--width=")
			n, nerr := strconv.Atoi(val)
			if nerr != nil || n < 20 {
				return 0, "", fmt.Errorf("--width must be an integer >= 20, got %q", val)
			}
			width = n
		case strings.HasPrefix(a, "-w="):
			val := strings.TrimPrefix(a, "-w=")
			n, nerr := strconv.Atoi(val)
			if nerr != nil || n < 20 {
				return 0, "", fmt.Errorf("-w must be an integer >= 20, got %q", val)
			}
			width = n
		case strings.HasPrefix(a, "-"):
			return 0, "", fmt.Errorf("unknown flag %q", a)
		default:
			if file != "" {
				return 0, "", fmt.Errorf("unexpected argument %q (only one file at a time)", a)
			}
			file = a
		}
	}
	return width, file, nil
}

func main() {
	width, file, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%serror:%s %v\n\n", fgRed+bold, reset, err)
		usage()
		os.Exit(1)
	}

	// Resolve render width: explicit flag > tty query > 80-col fallback.
	renderWidth := width
	if renderWidth == 0 {
		renderWidth = termWidth()
	}

	var src string
	if file != "" {
		data, rerr := os.ReadFile(file)
		if rerr != nil {
			fmt.Fprintf(os.Stderr, "%serror:%s cannot read %q: %v\n", fgRed+bold, reset, file, rerr)
			os.Exit(1)
		}
		src = string(data)
	} else {
		// No file — try stdin
		stat, serr := os.Stdin.Stat()
		if serr != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
			usage()
			os.Exit(1)
		}
		scanner := bufio.NewScanner(os.Stdin)
		var b strings.Builder
		for scanner.Scan() {
			b.WriteString(scanner.Text())
			b.WriteByte('\n')
		}
		src = b.String()
	}

	fmt.Print(render(src, renderWidth))
}
