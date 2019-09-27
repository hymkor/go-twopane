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

type LogEntry struct {
	Revision string `xml:"revision,attr"`
	Author   string `xml:"author"`
	Msg      string `xml:"msg"`
	contents string
}

type Log struct {
	LogEntry []LogEntry `xml:"logentry"`
}

func (this *LogEntry) Title(_ interface{}) string {
	if this.contents == "" {
		this.contents = this.Revision + " " + this.Msg
	}
	return this.contents
}

var testdata = []string{"test"}

func (this *LogEntry) Contents(_ interface{}) []string {
	return testdata
}

func makeRows() ([]twopane.Row, error) {
	bin, err := exec.Command("svn", "log", "-l", "100", "--xml").Output()
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
