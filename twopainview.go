package twopainview

import (
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-runewidth"
	"github.com/mattn/go-tty"
)

func getKey(tty1 *tty.TTY) (string, error) {
	clean, err := tty1.Raw()
	if err != nil {
		return "", err
	}
	defer clean()

	var buffer strings.Builder
	escape := false
	for {
		r, err := tty1.ReadRune()
		if err != nil {
			return "", err
		}
		if r == 0 {
			continue
		}
		buffer.WriteRune(r)
		if r == '\x1B' {
			escape = true
		}
		if !(escape && tty1.Buffered()) && buffer.Len() > 0 {
			return buffer.String(), nil
		}
	}
}

type Node interface {
	Title() string
	Contents() []string
}

const (
	CURSOR_OFF = "\x1B[?25l"
	CURSOR_ON  = "\x1B[?25h"
	BOLD_ON    = "\x1B[0;47;30m"
	BOLD_ON2   = "\x1B[0;1;7m"
	BOLD_OFF   = "\x1B[0m"
	UP_N       = "\x1B[%dA\r"
	ERASE_LINE = "\x1B[0K"
)

func view(nodes []Node, width, height, top, curr int, w io.Writer) int {
	newline := ""
	for i := 0; i < height; i++ {
		y := top + i
		if y >= len(nodes) {
			return i
		}
		fmt.Fprint(w, newline)
		newline = "\n"
		title := nodes[y].Title()
		if y == curr {
			fmt.Fprint(w, BOLD_ON)
		}
		fmt.Fprint(w, runewidth.Truncate(strings.TrimSpace(title), width-1, ""))
		if y == curr {
			fmt.Fprint(w, BOLD_OFF)
		}
		fmt.Fprint(w, ERASE_LINE)
	}
	return height
}

func Main(nodes []Node) error {
	tty1, err := tty.Open()
	if err != nil {
		return err
	}
	defer tty1.Close()

	width, height, err := tty1.Size()
	if err != nil {
		return err
	}
	top := 0
	current := 0
	viewheight := height / 2

	out := colorable.NewColorableStdout()
	fmt.Fprint(out, CURSOR_OFF)
	defer fmt.Fprint(out, CURSOR_ON)

	for {
		y := view(nodes, width, viewheight, top, current, out)
		fmt.Fprint(out, "\n\x1B[44;30m\x1B[0K\x1B[0m")

		for _, s := range nodes[current].Contents() {
			if y >= height-1 {
				break
			}
			fmt.Fprintln(out)
			y++
			fmt.Fprint(out, runewidth.Truncate(s, width-1, ""))
			fmt.Fprint(out, ERASE_LINE)
		}

		key, err := getKey(tty1)
		if err != nil {
			return err
		}
		switch key {
		case "j":
			if current < len(nodes)-1 {
				current++
				if current >= top+viewheight {
					top++
				}
			}
		case "k":
			if current > 0 {
				current--
				if current < top {
					top--
				}
			}
		case "q":
			fmt.Fprintln(out)
			return nil
		}
		fmt.Fprintf(out, UP_N, y)
	}
}
