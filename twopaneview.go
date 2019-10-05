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

const (
	_ANSI_CURSOR_OFF = "\x1B[?25l"
	_ANSI_CURSOR_ON  = "\x1B[?25h"
	_ANSI_BOLD       = "\x1B[0;41;37;1m"
	_ANSI_REVERSE    = "\x1B[0;7m"
	_ANSI_RESET      = "\x1B[0m"
	_ANSI_UP_N       = "\x1B[%dA\r"
	_ANSI_ERASE_LINE = "\x1B[0K"
)

const (
	_KEY_UP     = "\x1B[A"
	_KEY_ESC    = "\x1B"
	_KEY_DOWN   = "\x1B[B"
	_KEY_CTRL_N = "\x0E"
	_KEY_CTRL_P = "\x10"
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
			fmt.Fprint(&buffer, _ANSI_BOLD)
		}
		_, s := truncate(strings.TrimSpace(title), width-1)
		fmt.Fprint(&buffer, s)
		fmt.Fprint(&buffer, _ANSI_ERASE_LINE)
		if y == cursorY {
			fmt.Fprint(&buffer, _ANSI_RESET)
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

func expandContentsLine(src []string, max int) (dst []string) {
	for _, text := range src {
		text = textfilter(text)
		for {
			cutsize, line := truncate(text, max)
			dst = append(dst, line)
			if len(text) <= cutsize {
				break
			}
			text = text[cutsize:]
		}
	}
	return
}

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
	fmt.Fprint(v.Out, _ANSI_CURSOR_OFF)
	defer fmt.Fprint(v.Out, _ANSI_CURSOR_ON)

	if v.StatusLine == nil {
		v.StatusLine = strings.Repeat("=", width-1)
	}
	for {
		y := v.view(width, listHeight, headY, cursorY)
		fmt.Fprint(v.Out, "\n"+_ANSI_REVERSE)
		_, statusLine := truncate(fmt.Sprint(v.StatusLine), width-1)
		fmt.Fprint(v.Out, statusLine)
		fmt.Fprint(v.Out, _ANSI_ERASE_LINE+_ANSI_RESET)

		var index int
		if v.Reverse {
			index = len(v.Rows) - cursorY - 1
		} else {
			index = cursorY
		}
		for _, line := range expandContentsLine(v.Rows[index].Contents(v.X), width-1) {
			if y >= height-1 {
				goto viewEnd
			}
			fmt.Fprintln(v.Out)
			y++
			fmt.Fprint(v.Out, line)
			fmt.Fprint(v.Out, _ANSI_ERASE_LINE)
		}
		for y < height-1 {
			fmt.Fprintln(v.Out)
			fmt.Fprint(v.Out, _ANSI_ERASE_LINE)
			y++
		}
	viewEnd:
		key, err := getKey(tty1)
		if err != nil {
			return err
		}
		switch key {
		case "k", _KEY_CTRL_P, _KEY_UP:
			if cursorY > 0 {
				cursorY--
				if cursorY < headY {
					headY--
				}
			}
		case "q", _KEY_ESC:
			fmt.Fprint(v.Out, "\rQuit ? [Y/N] "+_ANSI_ERASE_LINE+"\r")
			if key, err := getKey(tty1); err == nil && key == "y" || key == "Y" {
				fmt.Fprintln(v.Out)
				return nil
			}
		case " ":
			v.cache = nil
			skip := height - (listHeight + 1)
			fmt.Fprintln(v.Out)
			contents := expandContentsLine(v.Rows[index].Contents(v.X), width-1)
			for i, text := range contents {
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
				if err != nil || key == _KEY_ESC {
					break
				}
				if key != " " {
					unGetKey = key
					break
				}
			}
			fallthrough
		case "j", _KEY_CTRL_N, _KEY_DOWN:
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
		fmt.Fprintf(v.Out, _ANSI_UP_N, y)
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
	fmt.Fprintf(p.Out, "\r%s%s", s, _ANSI_ERASE_LINE)
}
