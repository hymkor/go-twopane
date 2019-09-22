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

var unGetKey string

func getKey(tty1 *tty.TTY) (string, error) {
	if unGetKey != "" {
		rv := unGetKey
		unGetKey = ""
		return rv, nil
	}
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
	_CURSOR_OFF = "\x1B[?25l"
	_CURSOR_ON  = "\x1B[?25h"
	_BOLD_ON    = "\x1B[0;44;37;1m"
	_BOLD_OFF   = "\x1B[0m"
	_UP_N       = "\x1B[%dA\r"
	_ERASE_LINE = "\x1B[0K"
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

func view(rows []Row, width, height, head, cursor int, w io.Writer) int {
	newline := ""
	for i := 0; i < height; i++ {
		y := head + i
		if y >= len(rows) {
			return i
		}
		fmt.Fprint(w, newline)
		newline = "\n"
		title := rows[y].Title()
		if index := strings.IndexAny(title, "\r\n"); index >= 0 {
			title = title[:index]
		}
		if y == cursor {
			fmt.Fprint(w, _BOLD_ON)
		}
		fmt.Fprint(w, truncate(strings.TrimSpace(title), width-1))
		fmt.Fprint(w, _ERASE_LINE)
		if y == cursor {
			fmt.Fprint(w, _BOLD_OFF)
		}
	}
	return height
}

type View struct {
	Rows       []Row
	ViewHeight int
	Handler    func(*Param) bool
	Clear      bool // deprecated
	Out        io.Writer
}

type Param struct {
	*View
	Key    string
	Cursor int
	tty    *tty.TTY
}

func (v View) Run() error {
	tty1, err := tty.Open()
	if err != nil {
		return err
	}
	defer tty1.Close()

	width, height, err := tty1.Size()
	if err != nil {
		return err
	}
	head := 0
	cursor := 0
	if v.ViewHeight == 0 {
		v.ViewHeight = height / 2
	}
	listHeight := height - v.ViewHeight

	if v.Out == nil {
		v.Out = colorable.NewColorableStdout()
	}
	fmt.Fprint(v.Out, _CURSOR_OFF)
	defer fmt.Fprint(v.Out, _CURSOR_ON)

	hr := "\n\x1B[0;34;1m" + strings.Repeat("=", width-1) + "\x1B[0m"
	for {
		y := view(v.Rows, width, listHeight, head, cursor, v.Out)
		fmt.Fprint(v.Out, hr)

		for _, s := range v.Rows[cursor].Contents() {
			for {
				if y >= height-1 {
					goto viewEnd
				}
				fmt.Fprintln(v.Out)
				y++
				line := truncate(s, width-1)
				fmt.Fprint(v.Out, line)
				fmt.Fprint(v.Out, _ERASE_LINE)
				if len(s) <= len(line) {
					break
				}
				s = s[len(line):]
			}
		}
		for y < height-1 {
			fmt.Fprintln(v.Out)
			fmt.Fprint(v.Out, _ERASE_LINE)
			y++
		}
	viewEnd:
		key, err := getKey(tty1)
		if err != nil {
			return err
		}
		switch key {
		case "j", "\x0E", "\x1B[B":
			if cursor < len(v.Rows)-1 {
				cursor++
				if cursor >= head+listHeight {
					head++
				}
			}
		case "k", "\x10", "\x1B[A":
			if cursor > 0 {
				cursor--
				if cursor < head {
					head--
				}
			}
		case "q", "\x1B":
			fmt.Fprintln(v.Out)
			return nil
		default:
			if v.Handler != nil {
				param := &Param{
					View:   &v,
					Key:    key,
					Cursor: cursor,
					tty:    tty1,
				}
				if !v.Handler(param) {
					fmt.Fprintln(v.Out)
					return nil
				}
			}
		}
		fmt.Fprintf(v.Out, _UP_N, y)
	}
}

func (p *Param) GetKey() (string, error) {
	return getKey(p.tty)
}

func (p *Param) UnGetKey(s string) {
	unGetKey = s
}

func (p *Param) Message(s string) {
	fmt.Fprintf(p.Out, "\r%s%s", s, _ERASE_LINE)
}
