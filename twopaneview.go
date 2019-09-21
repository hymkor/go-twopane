package twopane

import (
	"fmt"
	"io"
	"strings"
	"unicode"

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

type Row interface {
	Title() string
	Contents() []string
}

const (
	CURSOR_OFF = "\x1B[?25l"
	CURSOR_ON  = "\x1B[?25h"
	BOLD_ON    = "\x1B[0;44;37;1m"
	BOLD_OFF   = "\x1B[0m"
	UP_N       = "\x1B[%dA\r"
	ERASE_LINE = "\x1B[0K"
)

func truncate(s string, max int) string {
	w := 0
	escape := false
	for i, c := range s {
		if escape {
			if unicode.IsLower(c) || unicode.IsUpper(c) {
				escape = false
			}
		} else if c == '\x1B' {
			escape = true
		} else {
			w1 := runewidth.RuneWidth(c)
			if w+w1 > max {
				return s[:i]
			}
			w += w1
		}
	}
	return s
}

func view(nodes []Row, width, height, top, curr int, w io.Writer) int {
	newline := ""
	for i := 0; i < height; i++ {
		y := top + i
		if y >= len(nodes) {
			return i
		}
		fmt.Fprint(w, newline)
		newline = "\n"
		title := nodes[y].Title()
		if index := strings.IndexAny(title, "\r\n"); index >= 0 {
			title = title[:index]
		}
		if y == curr {
			fmt.Fprint(w, BOLD_ON)
		}
		fmt.Fprint(w, truncate(strings.TrimSpace(title), width-1))
		fmt.Fprint(w, ERASE_LINE)
		if y == curr {
			fmt.Fprint(w, BOLD_OFF)
		}
	}
	return height
}

type Param struct {
	*View
	Key    string
	Cursor int
}

type View struct {
	Rows       []Row
	ViewHeight int
	Handler    func(*Param) bool
	Clear      bool
	Out        io.Writer
}

func (w View) Run() error {
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
	if w.ViewHeight == 0 {
		w.ViewHeight = height / 2
	}
	listHeight := height - w.ViewHeight

	if w.Out == nil {
		w.Out = colorable.NewColorableStdout()
	}
	fmt.Fprint(w.Out, CURSOR_OFF)
	defer fmt.Fprint(w.Out, CURSOR_ON)

	if w.Clear {
		fmt.Print("\x1B[2J\x1B[H")
	}

	hr := "\n\x1B[0;34;1m" + strings.Repeat("=", width-1) + "\x1B[0m"
	for {
		y := view(w.Rows, width, listHeight, top, current, w.Out)
		fmt.Fprint(w.Out, hr)

		for _, s := range w.Rows[current].Contents() {
			for {
				if y >= height-1 {
					goto viewEnd
				}
				fmt.Fprintln(w.Out)
				y++
				line := truncate(s, width-1)
				fmt.Fprint(w.Out, line)
				fmt.Fprint(w.Out, ERASE_LINE)
				if len(s) <= len(line) {
					break
				}
				s = s[len(line):]
			}
		}
		for y < height-1 {
			fmt.Fprintln(w.Out)
			fmt.Fprint(w.Out, ERASE_LINE)
			y++
		}
	viewEnd:
		key, err := getKey(tty1)
		if err != nil {
			return err
		}
		switch key {
		case "j", "\x0E", "\x1B[B":
			if current < len(w.Rows)-1 {
				current++
				if current >= top+listHeight {
					top++
				}
			}
		case "k", "\x10", "\x1B[A":
			if current > 0 {
				current--
				if current < top {
					top--
				}
			}
		case "q", "\x1B":
			fmt.Fprintln(w.Out)
			return nil
		default:
			if w.Handler != nil {
				param := &Param{
					View:   &w,
					Key:    key,
					Cursor: current,
				}
				if !w.Handler(param) {
					fmt.Fprintln(w.Out)
					return nil
				}
			}
		}
		fmt.Fprintf(w.Out, UP_N, y)
	}
}
