package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func usage() {
	fmt.Fprintf(os.Stderr, `%smdview%s — render Markdown in your terminal

%sUSAGE%s
  mdview [--width N] <file.md>
  cat README.md | mdview [--width N]

%sFLAGS%s
  -w, --width N   Override render width (default: auto-detect from terminal)
  -h, --help      Show this help

%sEXAMPLES%s
  mdview README.md
  mdview --width 120 README.md
  mdview CHANGELOG.md | less -R
  curl -s https://raw.githubusercontent.com/cli/cli/trunk/README.md | mdview

`,
		"\033[1m\033[95m", "\033[0m",
		"\033[1m\033[97m", "\033[0m",
		"\033[1m\033[97m", "\033[0m",
		"\033[1m\033[97m", "\033[0m",
	)
}

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
				return 0, "", fmt.Errorf("flag %q requires a number", a)
			}
			i++
			n, e := strconv.Atoi(args[i])
			if e != nil || n < 20 {
				return 0, "", fmt.Errorf("--width must be an integer ≥ 20")
			}
			width = n
		case strings.HasPrefix(a, "--width="):
			n, e := strconv.Atoi(strings.TrimPrefix(a, "--width="))
			if e != nil || n < 20 {
				return 0, "", fmt.Errorf("--width must be an integer ≥ 20")
			}
			width = n
		case strings.HasPrefix(a, "-w="):
			n, e := strconv.Atoi(strings.TrimPrefix(a, "-w="))
			if e != nil || n < 20 {
				return 0, "", fmt.Errorf("-w must be an integer ≥ 20")
			}
			width = n
		case strings.HasPrefix(a, "-"):
			return 0, "", fmt.Errorf("unknown flag %q", a)
		default:
			if file != "" {
				return 0, "", fmt.Errorf("only one file at a time")
			}
			file = a
		}
	}
	return
}

func main() {
	width, file, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[1m\033[31merror:\033[0m %v\n\n", err)
		usage()
		os.Exit(1)
	}

	w := width
	if w == 0 {
		w = termWidth()
	}

	var src string
	if file != "" {
		data, e := os.ReadFile(file)
		if e != nil {
			fmt.Fprintf(os.Stderr, "\033[1m\033[31merror:\033[0m cannot read %q: %v\n", file, e)
			os.Exit(1)
		}
		src = string(data)
	} else {
		stat, e := os.Stdin.Stat()
		if e != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
			usage()
			os.Exit(1)
		}
		var b strings.Builder
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			b.WriteString(sc.Text())
			b.WriteByte('\n')
		}
		src = b.String()
	}

	fmt.Print(render(src, w))
}
