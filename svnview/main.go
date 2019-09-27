package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/zetamatta/go-twopane"
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

func (this *LogEntry) Contents(_ interface{}) []string {
	if this.contents == nil {
		this.contents = strings.Split(this.Msg, "\n")
		this.contents = append(this.contents, "")
		for _, path1 := range this.Path {
			this.contents = append(this.contents,
				fmt.Sprintf("%s %s", path1.Action, path1.Text))
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
	statusLine := "=== [j] Up  [k] Down  [SPACE] git show  [q] Quit "
	statusLine = statusLine + strings.Repeat("=", 76-len(statusLine))

	return twopane.View{Rows: rows, StatusLine: statusLine}.Run()
}

func main() {
	if err := main1(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
