package twopane

import (
	"errors"
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

// Row is the interface for the element of list-view
type Row interface {
	Title(interface{}) string
	Contents(interface{}) []string
}

const (
	_CURSOR_OFF = "\x1B[?25l"
	_CURSOR_ON  = "\x1B[?25h"
	_BOLD_ON    = "\x1B[0;41;37;1m"
	_BOLD_OFF   = "\x1B[0m"
	_UP_N       = "\x1B[%dA\r"
	_ERASE_LINE = "\x1B[0K"
)

func truncate(s string, max int) (int, string) {
	w := 0
	escape := false
	var buffer strings.Builder
	for i, c := range s {
		if escape {
			if unicode.IsLower(c) || unicode.IsUpper(c) {
				escape = false
			}
			buffer.WriteRune(c)
		} else if c == '\x1B' {
			escape = true
			buffer.WriteRune(c)
		} else {
			var w1 int
			if c == '\t' {
				w1 = 8 - w%8
				if w+w1 > max {
					return i, buffer.String()
				}
				for i := 0; i < w1; i++ {
					buffer.WriteByte(' ')
				}
			} else {
				if runewidth.IsAmbiguousWidth(c) {
					w1 = 2
				} else {
					w1 = runewidth.RuneWidth(c)
				}
				if w+w1 > max {
					return i, buffer.String()
				}
				buffer.WriteRune(c)
			}
			w += w1
		}
	}
	return len(s), buffer.String()
}

func (v *View) view(width, height, headY, cursorY int) int {
	newline := ""
	if v.cache == nil {
		v.cache = map[int]string{}
	}
	for i := 0; i < height; i++ {
		y := headY + i
		if y >= len(v.Rows) {
			return i
		}
		fmt.Fprint(v.Out, newline)
		newline = "\n"

		var title string
		if v.Reverse {
			title = v.Rows[len(v.Rows)-y-1].Title(v.X)
		} else {
			title = v.Rows[y].Title(v.X)
		}
		title = textfilter(title)

		if index := strings.IndexAny(title, "\r\n"); index >= 0 {
			title = title[:index]
		}
		var buffer strings.Builder
		if y == cursorY {
			fmt.Fprint(&buffer, _BOLD_ON)
		}
		_, s := truncate(strings.TrimSpace(title), width-1)
		fmt.Fprint(&buffer, s)
		fmt.Fprint(&buffer, _ERASE_LINE)
		if y == cursorY {
			fmt.Fprint(&buffer, _BOLD_OFF)
		}
		b := buffer.String()
		if cache1, ok := v.cache[i]; !ok || cache1 != b {
			fmt.Fprint(v.Out, b)
			v.cache[i] = b
		}
	}
	return height
}

// View is the parameter for View.Run method.
type View struct {
	Rows       []Row
	ViewHeight int
	Handler    func(*Param) bool
	Clear      bool // deprecated
	Out        io.Writer
	Reverse    bool
	Cursor     int
	StatusLine interface{} // string or fmt.Stringer
	X          interface{} // is given to Title() and Contents() as first parameter
	cache      map[int]string
}

// Param is the parameters for the function called back from View.Run
type Param struct {
	*View
	Key    string
	Cursor int // index of cursor in View.Rows
	tty    *tty.TTY
	Width  int
	Height int
}

// ErrNoRows is the error when View.Rows has no rows.
var ErrNoRows = errors.New("no rows")

// Run shows View.Rows and wait and do your operations.
func (v View) Run() error {
	if len(v.Rows) <= 0 {
		return ErrNoRows
	}
	tty1, err := tty.Open()
	if err != nil {
		return err
	}
	quit := make(chan struct{})
	ws := tty1.SIGWINCH()
	go func() {
		for {
			select {
			case <-quit:
				return
			case <-ws:
			}
		}
	}()

	defer func() {
		tty1.Close()
		quit <- struct{}{}
	}()

	width, height, err := tty1.Size()
	if err != nil {
		return err
	}
	headY := 0
	cursorY := v.Cursor
	if v.ViewHeight == 0 {
		v.ViewHeight = height / 2
	}
	listHeight := height - v.ViewHeight

	if v.Out == nil {
		v.Out = colorable.NewColorableStdout()
	}
	fmt.Fprint(v.Out, _CURSOR_OFF)
	defer fmt.Fprint(v.Out, _CURSOR_ON)

	if v.StatusLine == nil {
		v.StatusLine = strings.Repeat("=", width-1)
	}
	for {
		y := v.view(width, listHeight, headY, cursorY)
		fmt.Fprint(v.Out, "\n\x1B[0;7m")
		_, statusLine := truncate(fmt.Sprint(v.StatusLine), width-1)
		fmt.Fprint(v.Out, statusLine)
		fmt.Fprint(v.Out, _ERASE_LINE+"\x1B[0m")

		var index int
		if v.Reverse {
			index = len(v.Rows) - cursorY - 1
		} else {
			index = cursorY
		}
		for _, _s := range v.Rows[index].Contents(v.X) {
			s := textfilter(_s)
			for {
				if y >= height-1 {
					goto viewEnd
				}
				fmt.Fprintln(v.Out)
				y++
				cutsize, line := truncate(s, width-1)
				fmt.Fprint(v.Out, line)
				fmt.Fprint(v.Out, _ERASE_LINE)
				if len(s) <= cutsize {
					break
				}
				s = s[cutsize:]
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
		case "k", "\x10", "\x1B[A":
			if cursorY > 0 {
				cursorY--
				if cursorY < headY {
					headY--
				}
			}
		case "q", "\x1B":
			fmt.Fprint(v.Out, "\rQuit ? [Y/N] "+_ERASE_LINE+"\r")
			if key, err := getKey(tty1); err == nil && key == "y" || key == "Y" {
				fmt.Fprintln(v.Out)
				return nil
			}
		case " ":
			v.cache = nil
			skip := height - (listHeight + 1)
			fmt.Fprintln(v.Out)
			contents := v.Rows[index].Contents(v.X)
			for i, _text := range contents {
				text := textfilter(_text)
				if i >= skip {
					fmt.Fprintln(v.Out, text)
				}
				if ((i + 1) % height) == 0 {
					fmt.Fprint(v.Out, "\r[more]")
					if key, err := getKey(tty1); err != nil || key == "q" || key == "\x1B" {
						goto done
					}
					fmt.Fprint(v.Out, "\r      \r")
				}
			}
			if len(contents) >= skip {
				fmt.Fprint(v.Out, "[next]")
				key, err := getKey(tty1)
				if err != nil || key == "\x1B" {
					break
				}
				if key != " " {
					unGetKey = key
					break
				}
			}
			fallthrough
		case "j", "\x0E", "\x1B[B":
			if cursorY < len(v.Rows)-1 {
				cursorY++
				if cursorY >= headY+listHeight {
					headY++
				}
			}
		default:
			v.cache = nil
			if v.Handler != nil {
				param := &Param{
					View:   &v,
					Key:    key,
					Cursor: index,
					tty:    tty1,
					Width:  width,
					Height: height,
				}
				if !v.Handler(param) {
					fmt.Fprintln(v.Out)
					return nil
				}
				if v.Reverse {
					cursorY = len(v.Rows) - param.Cursor - 1
				} else {
					cursorY = param.Cursor
				}
				if cursorY < headY {
					headY = cursorY
				} else if cursorY >= headY+listHeight {
					headY = cursorY - listHeight + 1
				}
			}
		}
	done:
		fmt.Fprintf(v.Out, _UP_N, y)
	}
}

// GetKey gets user typed key.
func (p *Param) GetKey() (string, error) {
	return getKey(p.tty)
}

// UnGetKey sets the value the next called GetKey returns.
func (p *Param) UnGetKey(s string) {
	unGetKey = s
}

// Message shows string `s` on the bottom line of the screen.
func (p *Param) Message(s string) {
	fmt.Fprintf(p.Out, "\r%s%s", s, _ERASE_LINE)
}
