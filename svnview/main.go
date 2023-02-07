package main

import (
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/text/transform"

	"github.com/nyaosorg/go-windows-mbcs"

	"github.com/hymkor/go-twopane"
)

type Path struct {
	Action string `xml:"action,attr"`
	Text   string `xml:",chardata"`
}

type LogEntry struct {
	Revision string `xml:"revision,attr"`
	Author   string `xml:"author"`
	Msg      string `xml:"msg"`
	Path     []Path `xml:"paths>path"`
	title    string
	contents []string
}

type Log struct {
	LogEntry []LogEntry `xml:"logentry"`
}

func (this *LogEntry) Title(_ interface{}) string {
	if this.title == "" {
		this.title = fmt.Sprintf("r%s %.8s %s", this.Revision, this.Author, this.Msg)
	}
	return this.title
}

func (this *LogEntry) diff(contents []string) ([]string, error) {
	rev, err := strconv.Atoi(this.Revision)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("svn", "diff", fmt.Sprintf("-r%d:%d", rev-1, rev))
	in, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer in.Close()

	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	defer cmd.Wait()

	sc := bufio.NewScanner(transform.NewReader(in, mbcs.Decoder{CP: mbcs.ACP}))
	for sc.Scan() {
		contents = append(contents, sc.Text())
	}
	err = sc.Err()
	if err != io.EOF {
		err = nil
	}
	return contents, err
}

func (this *LogEntry) Contents(_ interface{}) []string {
	if this.contents == nil {
		this.contents = strings.Split(this.Msg, "\n")
		this.contents = append(this.contents, "")
		for _, path1 := range this.Path {
			this.contents = append(this.contents,
				fmt.Sprintf("%s %s", path1.Action, path1.Text))
		}
		contents, err := this.diff(this.contents)
		if err != nil {
			this.contents = append(this.contents, err.Error())
		} else {
			this.contents = contents
		}
	}
	return this.contents
}

func makeRows() ([]twopane.Row, error) {
	bin, err := exec.Command("svn", "log", "-v", "-l", "100", "--xml").Output()
	if err != nil {
		return nil, err
	}
	if bin == nil {
		return nil, errors.New("svn log output is nil")
	}
	var logdata Log
	if err := xml.Unmarshal(bin, &logdata); err != nil {
		return nil, err
	}
	rows := make([]twopane.Row, 0, len(logdata.LogEntry))
	for _, logentry := range logdata.LogEntry {
		tmp := logentry
		rows = append(rows, &tmp)
	}
	return rows, nil
}

func main1() error {
	rows, err := makeRows()
	if err != nil {
		return err
	}
	statusLine := "=== [j] Up  [k] Down  [SPACE] More [q] Quit "
	statusLine = statusLine + strings.Repeat("=", 76-len(statusLine))

	return twopane.View{Rows: rows, StatusLine: statusLine}.Run()
}

func main() {
	if err := main1(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
