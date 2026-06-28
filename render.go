package main

import (
        "fmt"
        "regexp"
        "strings"
        "syscall"
        "unicode/utf8"
        "unsafe"
)

// ANSI codes

const (
        reset     = "\033[0m"
        bold      = "\033[1m"
        italic    = "\033[3m"
        underline = "\033[4m"
        strike    = "\033[9m"

        fgRed     = "\033[31m"
        fgGreen   = "\033[32m"
        fgYellow  = "\033[33m"
        fgBlue    = "\033[34m"
        fgMagenta = "\033[35m"
        fgCyan    = "\033[36m"

        fgBrightBlack   = "\033[90m"
        fgBrightGreen   = "\033[92m"
        fgBrightYellow  = "\033[93m"
        fgBrightBlue    = "\033[94m"
        fgBrightMagenta = "\033[95m"
        fgBrightCyan    = "\033[96m"
        fgBrightWhite   = "\033[97m"

        bgBrightBlack = "\033[100m"
)

// Terminal width

type winsize struct {
        Row, Col, Xpixel, Ypixel uint16
}

func termWidth() int {
        for _, fd := range []uintptr{
                uintptr(syscall.Stdout),
                uintptr(syscall.Stderr),
                uintptr(syscall.Stdin),
        } {
                ws := &winsize{}
                rc, _, _ := syscall.Syscall(syscall.SYS_IOCTL, fd,
                        syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(ws)))
                if rc == 0 && ws.Col > 0 {
                        return int(ws.Col)
                }
        }
        return 80
}

func render(src string, width int) string {
        if width < 20 {
                width = 20
        }
        r := newRenderer(width)
        parse(src, r)
        return r.out.String()
}

// String helpers

func rep(s string, n int) string {
        if n <= 0 {
                return ""
        }
        return strings.Repeat(s, n)
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func vlen(s string) int {
        return utf8.RuneCountInString(ansiRe.ReplaceAllString(s, ""))
}

func padR(s string, n int) string {
        l := vlen(s)
        if l >= n {
                return s
        }
        return s + rep(" ", n-l)
}

func centerStr(s string, w int) string {
        l := vlen(s)
        if l >= w {
                return s
        }
        pad := (w - l) / 2
        return rep(" ", pad) + s
}

func minInt(a, b int) int {
        if a < b {
                return a
        }
        return b
}

// Word wrapping

// hardBreakWord splits a single word into chunks of ≤ maxW runes.
func hardBreakWord(w string, maxW int, out *[]string) {
        runes := []rune(w)
        for len(runes) > 0 {
                end := minInt(maxW, len(runes))
                *out = append(*out, string(runes[:end]))
                runes = runes[end:]
        }
}

// wrapWords wraps text into lines of ≤ maxW visible chars.
// Words wider than maxW are hard-broken by character.
func wrapWords(text string, maxW int) []string {
        if maxW <= 0 {
                return []string{text}
        }
        words := strings.Fields(text)
        if len(words) == 0 {
                return []string{""}
        }
        var lines []string
        cur := ""
        for _, w := range words {
                wl := vlen(w)
                switch {
                case cur == "" && wl > maxW:
                        hardBreakWord(w, maxW, &lines)
                case cur == "":
                        cur = w
                case vlen(cur)+1+wl <= maxW:
                        cur += " " + w
                default:
                        lines = append(lines, cur)
                        cur = ""
                        if wl > maxW {
                                hardBreakWord(w, maxW, &lines)
                        } else {
                                cur = w
                        }
                }
        }
        if cur != "" {
                lines = append(lines, cur)
        }
        return lines
}

// Inline markdown renderer

var (
        reBoldItalic = regexp.MustCompile(`\*{3}(.+?)\*{3}|_{3}(.+?)_{3}`)
        reBold       = regexp.MustCompile(`\*{2}(.+?)\*{2}|_{2}(.+?)_{2}`)
        reItalic     = regexp.MustCompile(`\*([^*\n]+?)\*|_([^_\n]+?)_`)
        reStrike     = regexp.MustCompile(`~~(.+?)~~`)
        reCode       = regexp.MustCompile("`([^`]+)`")
        reLink       = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
        reImage      = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
        reAutolink   = regexp.MustCompile(`<(https?://[^>]+)>`)
)

func renderInline(s string) string {
        s = reImage.ReplaceAllStringFunc(s, func(m string) string {
                sub := reImage.FindStringSubmatch(m)
                alt := sub[1]
                if alt == "" {
                        alt = "image"
                }
                return fgBrightBlack + "[" + alt + "]" + reset
        })
        s = reLink.ReplaceAllStringFunc(s, func(m string) string {
                sub := reLink.FindStringSubmatch(m)
                return bold + fgBrightBlue + sub[1] + reset +
                        fgBrightBlack + " (" + sub[2] + ")" + reset
        })
        s = reAutolink.ReplaceAllStringFunc(s, func(m string) string {
                sub := reAutolink.FindStringSubmatch(m)
                return fgBrightBlue + underline + sub[1] + reset
        })
        s = reCode.ReplaceAllStringFunc(s, func(m string) string {
                sub := reCode.FindStringSubmatch(m)
                return bgBrightBlack + fgBrightWhite + " " + sub[1] + " " + reset
        })
        s = reBoldItalic.ReplaceAllStringFunc(s, func(m string) string {
                sub := reBoldItalic.FindStringSubmatch(m)
                t := sub[1]
                if t == "" {
                        t = sub[2]
                }
                return bold + italic + t + reset
        })
        s = reBold.ReplaceAllStringFunc(s, func(m string) string {
                sub := reBold.FindStringSubmatch(m)
                t := sub[1]
                if t == "" {
                        t = sub[2]
                }
                return bold + t + reset
        })
        s = reItalic.ReplaceAllStringFunc(s, func(m string) string {
                sub := reItalic.FindStringSubmatch(m)
                t := sub[1]
                if t == "" {
                        t = sub[2]
                }
                return italic + t + reset
        })
        s = reStrike.ReplaceAllStringFunc(s, func(m string) string {
                sub := reStrike.FindStringSubmatch(m)
                return strike + fgBrightBlack + sub[1] + reset
        })
        return s
}

// HTML center-block processing

var (
        reDivCenter    = regexp.MustCompile(`(?i)<div[^>]+align\s*=\s*["']?center["']?[^>]*>`)
        rePCenter      = regexp.MustCompile(`(?i)<p[^>]+align\s*=\s*["']?center["']?[^>]*>`)
        rePClose       = regexp.MustCompile(`(?i)</p>`)
        reDivClose     = regexp.MustCompile(`(?i)</div>`)
        reHTMLBr       = regexp.MustCompile(`(?i)<br\s*/?>`)
        reHTMLImgAlt   = regexp.MustCompile(`(?i)<img\b[^>]*\balt\s*=\s*["']([^"']*)["'][^>]*/?>`)
        reHTMLImgNoAlt = regexp.MustCompile(`(?i)<img\b[^>]*/?>`)
        reHTMLHref     = regexp.MustCompile(`(?i)<a\b[^>]*>(.*?)</a\s*>`)
        reHTMLHeading  = regexp.MustCompile(`(?i)<h([1-6])\b[^>]*>(.*?)</h[1-6]>`)
        reHTMLBold     = regexp.MustCompile(`(?i)<(?:b|strong)\b[^>]*>(.*?)</(?:b|strong)>`)
        reHTMLEm       = regexp.MustCompile(`(?i)<(?:i|em)\b[^>]*>(.*?)</(?:i|em)>`)
        reHTMLComment  = regexp.MustCompile(`<!--.*?-->`)
        reHTMLAnyTag   = regexp.MustCompile(`<[^>]+>`)
        reHTMLEntity   = regexp.MustCompile(`&(?:[a-zA-Z]+|#[0-9]+);`)
)

var htmlEntities = map[string]string{
        "&amp;":    "&",
        "&lt;":     "<",
        "&gt;":     ">",
        "&quot;":   `"`,
        "&apos;":   "'",
        "&nbsp;":   " ",
        "&mdash;":  "—",
        "&ndash;":  "–",
        "&hellip;": "…",
        "&copy;":   "©",
        "&reg;":    "®",
        "&trade;":  "™",
}

func decodeEntities(s string) string {
        return reHTMLEntity.ReplaceAllStringFunc(s, func(e string) string {
                if v, ok := htmlEntities[e]; ok {
                        return v
                }
                return e
        })
}

// processHTMLLine converts one line of HTML (inside a center block) to text
// segments. "" in the slice means emit a blank line; nil means skip entirely.
func processHTMLLine(raw string) []string {
        line := strings.TrimSpace(raw)
        if line == "" {
                return []string{""}
        }

        line = reHTMLComment.ReplaceAllString(line, "")

        hasBR := reHTMLBr.MatchString(line)
        line = reHTMLBr.ReplaceAllString(line, " ")

        // ATX markdown heading inside a center block: # Heading, ## Heading, …
        if strings.HasPrefix(line, "#") {
                lvl := 0
                for _, ch := range line {
                        if ch == '#' {
                                lvl++
                        } else {
                                break
                        }
                }
                if lvl <= 6 && len(line) > lvl && line[lvl] == ' ' {
                        text := strings.TrimSpace(line[lvl+1:])
                        if text == "" {
                                return nil
                        }
                        var style string
                        switch lvl {
                        case 1:
                                style = bold + fgBrightMagenta
                        case 2:
                                style = bold + fgBrightCyan
                        case 3:
                                style = bold + fgBrightYellow
                        default:
                                style = bold + fgBrightWhite
                        }
                        return []string{style + text + reset}
                }
        }

        // <h1>…</h6> — styled heading text
        if m := reHTMLHeading.FindStringSubmatch(line); m != nil {
                inner := reHTMLAnyTag.ReplaceAllString(m[2], "")
                inner = decodeEntities(strings.TrimSpace(inner))
                if inner == "" {
                        return nil
                }
                var style string
                switch m[1] {
                case "1":
                        style = bold + fgBrightMagenta
                case "2":
                        style = bold + fgBrightCyan
                case "3":
                        style = bold + fgBrightYellow
                default:
                        style = bold + fgBrightWhite
                }
                return []string{style + inner + reset}
        }

        // <img alt="…"> → [alt], <img> without alt → omit
        line = reHTMLImgAlt.ReplaceAllStringFunc(line, func(m string) string {
                sub := reHTMLImgAlt.FindStringSubmatch(m)
                alt := strings.TrimSpace(sub[1])
                if alt != "" {
                        return fgBrightBlack + "[" + alt + "]" + reset
                }
                return ""
        })
        line = reHTMLImgNoAlt.ReplaceAllString(line, "")

        // <a href="…">content</a> → coloured link text
        line = reHTMLHref.ReplaceAllStringFunc(line, func(m string) string {
                sub := reHTMLHref.FindStringSubmatch(m)
                inner := strings.TrimSpace(reHTMLAnyTag.ReplaceAllString(sub[1], ""))
                if inner == "" {
                        return ""
                }
                return bold + fgBrightBlue + inner + reset
        })

        // <b>/<strong> and <i>/<em>
        line = reHTMLBold.ReplaceAllStringFunc(line, func(m string) string {
                sub := reHTMLBold.FindStringSubmatch(m)
                return bold + sub[1] + reset
        })
        line = reHTMLEm.ReplaceAllStringFunc(line, func(m string) string {
                sub := reHTMLEm.FindStringSubmatch(m)
                return italic + sub[1] + reset
        })

        // Strip remaining tags
        line = reHTMLAnyTag.ReplaceAllString(line, "")
        line = decodeEntities(strings.TrimSpace(line))

        if line == "" && !hasBR {
                return nil
        }
        var out []string
        if line != "" {
                out = append(out, line)
        }
        if hasBR {
                out = append(out, "")
        }
        return out
}

// Renderer

type renderer struct {
        w   int
        out strings.Builder
}

func newRenderer(w int) *renderer { return &renderer{w: w} }

func (r *renderer) nl()           { r.out.WriteByte('\n') }
func (r *renderer) line(s string) { r.out.WriteString(s); r.out.WriteByte('\n') }

func (r *renderer) heading(level int, text string) {
        t := renderInline(text)
        r.nl()
        switch level {
        case 1:
                r.line(bold + fgBrightMagenta + "██" + reset + " " + bold + fgBrightWhite + strings.ToUpper(t) + reset)
                r.line(fgBrightMagenta + rep("─", r.w) + reset)
        case 2:
                r.line(bold + fgBrightCyan + "## " + t + reset)
                r.line(fgCyan + rep("─", r.w/2) + reset)
        case 3:
                r.line(bold + fgBrightYellow + "### " + t + reset)
        case 4:
                r.line(bold + fgYellow + "#### " + t + reset)
        case 5:
                r.line(bold + fgBrightBlack + "##### " + t + reset)
        default:
                r.line(bold + fgBrightBlack + "###### " + t + reset)
        }
}

func (r *renderer) hr() {
        r.nl()
        r.line(fgBrightBlack + rep("─", r.w) + reset)
        r.nl()
}

func (r *renderer) codeBlock(lang string, lines []string) {
        r.nl()
        hdr := bgBrightBlack + fgBrightWhite + bold
        if lang != "" {
                hdr += "  " + strings.ToUpper(lang) + "  "
        } else {
                hdr += "  CODE  "
        }
        hdr += reset
        r.line(hdr)
        bar := fgBrightBlack + "│" + reset
        for _, l := range lines {
                r.line(bar + " " + fgBrightGreen + l + reset)
        }
        r.line(fgBrightBlack + rep("─", r.w) + reset)
        r.nl()
}

func (r *renderer) blockquote(lines []string) {
        r.nl()
        for _, l := range lines {
                r.line(fgBrightBlack + "▌ " + reset + italic + fgBrightWhite + renderInline(l) + reset)
        }
        r.nl()
}

func (r *renderer) listItem(depth int, ord bool, idx int, text string) {
        indent := rep("  ", depth)
        var bullet string
        if ord {
                if depth == 0 {
                        bullet = fgBrightMagenta + fmt.Sprintf("%d.", idx) + reset
                } else {
                        bullet = fgBrightBlack + fmt.Sprintf("%d.", idx) + reset
                }
        } else {
                if depth == 0 {
                        bullet = fgBrightMagenta + "•" + reset
                } else {
                        bullet = fgBrightBlack + "◦" + reset
                }
        }
        r.line(indent + bullet + " " + renderInline(text))
}

func (r *renderer) paragraph(lines []string) {
        text := renderInline(strings.Join(lines, " "))
        cur := ""
        for _, w := range strings.Fields(text) {
                if cur == "" {
                        cur = w
                } else if vlen(cur)+1+vlen(w) <= r.w {
                        cur += " " + w
                } else {
                        r.line(cur)
                        cur = w
                }
        }
        if cur != "" {
                r.line(cur)
        }
        r.nl()
}

// Table rendering

// shrinkCols reduces column widths proportionally until the table fits in limit.
// Table wire width = 1 + Σ(colW + 3)  (leading │, then space+content+space+│ per col)
func shrinkCols(widths []int, limit int) []int {
        n := len(widths)
        if n == 0 {
                return widths
        }
        out := make([]int, n)
        copy(out, widths)

        const minW = 4

        tableW := func() int {
                s := 1
                for _, w := range out {
                        s += w + 3
                }
                return s
        }

        if tableW() <= limit {
                return out
        }

        // Total content budget
        budget := limit - 1 - n*3
        if budget < n*minW {
                budget = n * minW
        }

        total := 0
        for _, w := range out {
                total += w
        }
        if total == 0 {
                return out
        }

        // Proportional distribution
        remaining := budget
        for i := range out {
                share := out[i] * budget / total
                if share < minW {
                        share = minW
                }
                out[i] = share
                remaining -= share
        }
        // Give remainder left-to-right
        for i := range out {
                if remaining <= 0 {
                        break
                }
                out[i]++
                remaining--
        }
        // Safety trim (rounding overshoot)
        for tableW() > limit {
                best := -1
                for i, w := range out {
                        if w > minW && (best == -1 || w > out[best]) {
                                best = i
                        }
                }
                if best == -1 {
                        break
                }
                out[best]--
        }
        return out
}

func (r *renderer) table(rows []string) {
        if len(rows) == 0 {
                return
        }

        parseCells := func(row string) []string {
                row = strings.Trim(row, "| \t")
                parts := strings.Split(row, "|")
                out := make([]string, len(parts))
                for i, p := range parts {
                        out[i] = strings.TrimSpace(p)
                }
                return out
        }

        sepRe := regexp.MustCompile(`^[\s|:\-]+$`)
        hasSep := len(rows) >= 2 && sepRe.MatchString(rows[1])

        var allRows [][]string
        for i, row := range rows {
                if hasSep && i == 1 {
                        continue
                }
                allRows = append(allRows, parseCells(row))
        }
        if len(allRows) == 0 {
                return
        }

        // Natural column widths
        numCols := 0
        for _, row := range allRows {
                if len(row) > numCols {
                        numCols = len(row)
                }
        }
        colW := make([]int, numCols)
        for _, row := range allRows {
                for ci, cell := range row {
                        if ci < numCols && len(cell) > colW[ci] {
                                colW[ci] = len(cell)
                        }
                }
        }

        // Shrink to fit terminal
        colW = shrinkCols(colW, r.w)

        hline := func(l, m, ri, fill string) string {
                parts := make([]string, numCols)
                for i, w := range colW {
                        parts[i] = rep(fill, w+2)
                }
                return fgBrightBlack + l + strings.Join(parts, m) + ri + reset
        }

        r.nl()
        r.line(hline("┌", "┬", "┐", "─"))

        for ri, row := range allRows {
                // Wrap each cell to its column width
                wrapped := make([][]string, numCols)
                maxL := 1
                for ci := 0; ci < numCols; ci++ {
                        raw := ""
                        if ci < len(row) {
                                raw = row[ci]
                        }
                        // Wrap on plain text so inline ANSI spans (e.g. `code`) are
                        // never split by strings.Fields. Apply renderInline afterwards.
                        wlines := wrapWords(raw, colW[ci])
                        for j, l := range wlines {
                                wlines[j] = renderInline(l)
                        }
                        wrapped[ci] = wlines
                        if len(wlines) > maxL {
                                maxL = len(wlines)
                        }
                }

                for li := 0; li < maxL; li++ {
                        out := fgBrightBlack + "│" + reset
                        for ci := 0; ci < numCols; ci++ {
                                cell := ""
                                if li < len(wrapped[ci]) {
                                        cell = wrapped[ci][li]
                                }
                                if ri == 0 {
                                        // Header row: strip existing ANSI then apply bold+white
                                        cell = bold + fgBrightWhite + ansiRe.ReplaceAllString(cell, "") + reset
                                }
                                out += " " + padR(cell, colW[ci]) + " " + fgBrightBlack + "│" + reset
                        }
                        r.line(out)
                }

                if ri == 0 {
                        r.line(hline("├", "┼", "┤", "─"))
                }
        }

        r.line(hline("└", "┴", "┘", "─"))
        r.nl()
}

// centerBlock emits a slice of raw HTML lines centered in the terminal.
func (r *renderer) centerBlock(lines []string) {
        r.nl()
        for _, raw := range lines {
                segs := processHTMLLine(raw)
                for _, seg := range segs {
                        if seg == "" {
                                r.nl()
                        } else {
                                seg = renderInline(seg)
                                r.line(centerStr(seg, r.w))
                        }
                }
        }
        r.nl()
}

// Line classifier

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
        kind  lineKind
        raw   string
        level int
        depth int
        ord   bool
        index int
}

func classify(s string) parsedLine {
        p := parsedLine{raw: s}

        trimmed := strings.TrimSpace(s)

        if trimmed == "" {
                p.kind = kindBlank
                return p
        }
        if strings.HasPrefix(trimmed, "<!--") {
                p.kind = kindHTMLComment
                return p
        }
        if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
                p.kind = kindFence
                return p
        }

        // ATX heading
        if strings.HasPrefix(trimmed, "#") {
                lvl := 0
                for _, ch := range trimmed {
                        if ch == '#' {
                                lvl++
                        } else {
                                break
                        }
                }
                if lvl <= 6 && len(trimmed) > lvl && trimmed[lvl] == ' ' {
                        p.kind = kindHeading
                        p.level = lvl
                        p.raw = strings.TrimSpace(trimmed[lvl+1:])
                        return p
                }
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

        // Table
        if strings.Contains(trimmed, "|") {
                p.kind = kindTable
                return p
        }

        p.kind = kindParagraph
        return p
}

// Main parse loop

func parse(src string, r *renderer) {
        rawLines := strings.Split(src, "\n")
        // Trim trailing empty line that Split adds for a trailing newline
        if len(rawLines) > 0 && rawLines[len(rawLines)-1] == "" {
                rawLines = rawLines[:len(rawLines)-1]
        }

        // Block state
        inFence := false
        fenceLang := ""
        var fenceLines []string

        inBQ := false
        var bqLines []string

        var paraLines []string

        inTable := false
        var tableRows []string

        inCenter := false
        divDepth := 0
        var centerLines []string

        flush := func() {
                if len(paraLines) > 0 {
                        r.paragraph(paraLines)
                        paraLines = nil
                }
                if len(bqLines) > 0 {
                        r.blockquote(bqLines)
                        bqLines = nil
                        inBQ = false
                }
                if len(tableRows) > 0 {
                        r.table(tableRows)
                        tableRows = nil
                        inTable = false
                }
        }

        for i, raw := range rawLines {
                trimmed := strings.TrimSpace(raw)

                // Center-block: <div align="center">
                if !inFence {
                        // Strip backtick spans before tag-matching so `<div align="center">`
                        // inside table cells or code spans does not open a center block.
                        htmlStripped := reCode.ReplaceAllString(trimmed, "")
                        if reDivCenter.MatchString(htmlStripped) || rePCenter.MatchString(htmlStripped) {
                                flush()
                                if inCenter {
                                        divDepth++
                                        centerLines = append(centerLines, raw)
                                } else {
                                        inCenter = true
                                        divDepth = 1
                                }
                                continue
                        }
                        if inCenter {
                                isClose := reDivClose.MatchString(trimmed) || rePClose.MatchString(trimmed)
                                if isClose {
                                        divDepth--
                                        if divDepth <= 0 {
                                                r.centerBlock(centerLines)
                                                centerLines = nil
                                                inCenter = false
                                        } else {
                                                centerLines = append(centerLines, raw)
                                        }
                                } else {
                                        centerLines = append(centerLines, raw)
                                }
                                continue
                        }
                }

                // Setext headings: look one line ahead
                if !inFence && i+1 < len(rawLines) && trimmed != "" {
                        next := strings.TrimSpace(rawLines[i+1])
                        allEq := regexp.MustCompile(`^=+$`)
                        allDash := regexp.MustCompile(`^-{2,}$`)
                        if allEq.MatchString(next) {
                                flush()
                                r.heading(1, trimmed)
                                continue
                        }
                        if allDash.MatchString(next) && !strings.HasPrefix(trimmed, "#") {
                                flush()
                                r.heading(2, trimmed)
                                continue
                        }
                }
                // Skip setext underline itself
                if !inFence && i > 0 {
                        prev := strings.TrimSpace(rawLines[i-1])
                        allEqRe := regexp.MustCompile(`^=+\s*$`)
                        allDashRe := regexp.MustCompile(`^-{2,}\s*$`)
                        if (allEqRe.MatchString(trimmed) || allDashRe.MatchString(trimmed)) &&
                                prev != "" && !strings.HasPrefix(prev, "#") {
                                continue
                        }
                }

                // Fenced code block
                if inFence {
                        t2 := strings.TrimLeft(raw, " \t")
                        if strings.HasPrefix(t2, "```") || strings.HasPrefix(t2, "~~~") {
                                r.codeBlock(fenceLang, fenceLines)
                                fenceLines = nil
                                fenceLang = ""
                                inFence = false
                        } else {
                                fenceLines = append(fenceLines, raw)
                        }
                        continue
                }

                p := classify(raw)

                switch p.kind {
                case kindFence:
                        flush()
                        inFence = true
                        fenceLang = strings.TrimSpace(strings.TrimLeft(raw, " \t~`"))

                case kindBlank:
                        flush()
                        r.nl()

                case kindHTMLComment:
                        // skip

                case kindHeading:
                        flush()
                        r.heading(p.level, p.raw)

                case kindHR:
                        flush()
                        r.hr()

                case kindBlockquote:
                        if len(paraLines) > 0 {
                                r.paragraph(paraLines)
                                paraLines = nil
                        }
                        if len(tableRows) > 0 {
                                r.table(tableRows)
                                tableRows = nil
                                inTable = false
                        }
                        inBQ = true
                        bqLines = append(bqLines, p.raw)

                case kindListItem:
                        flush()
                        r.listItem(p.depth, p.ord, p.index, p.raw)

                case kindTable:
                        if len(paraLines) > 0 {
                                r.paragraph(paraLines)
                                paraLines = nil
                        }
                        if inBQ {
                                r.blockquote(bqLines)
                                bqLines = nil
                                inBQ = false
                        }
                        inTable = true
                        tableRows = append(tableRows, raw)

                case kindIndentedCode:
                        flush()
                        r.codeBlock("", []string{p.raw})

                case kindParagraph:
                        if inBQ {
                                r.blockquote(bqLines)
                                bqLines = nil
                                inBQ = false
                        }
                        if inTable {
                                r.table(tableRows)
                                tableRows = nil
                                inTable = false
                        }
                        paraLines = append(paraLines, trimmed)
                }
        }

        // Flush any remaining open blocks
        flush()
        if inFence && len(fenceLines) > 0 {
                r.codeBlock(fenceLang, fenceLines)
        }
        if inCenter && len(centerLines) > 0 {
                r.centerBlock(centerLines)
        }
}
