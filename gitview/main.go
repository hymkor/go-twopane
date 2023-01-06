package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hymkor/go-twopane"
)

type Row struct {
	commit   string
	title    string
	contents []string
}

func (this *Row) Title(_ interface{}) string {
	return this.title
}

func fetchOutput(cmd *exec.Cmd, callback func(text string)) error {
	in, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer in.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}
	defer cmd.Wait()

	sc := bufio.NewScanner(in)
	for sc.Scan() {
		callback(sc.Text())
	}
	return nil
}

func (this *Row) Contents(_ interface{}) []string {
	if this.contents == nil {
		cmd := exec.Command("git", "show", "--color", this.commit)
		err := fetchOutput(cmd, func(text string) {
			this.contents = append(this.contents, textfilter(text))
		})
		if err != nil {
			this.contents = []string{err.Error()}
		}
	}
	return this.contents
}

func makeRows() ([]twopane.Row, error) {
	rows := []twopane.Row{}
	cmd := exec.Command("git", "log", "-n", "100", "--pretty=format:%H\t%h %s")

	err := fetchOutput(cmd, func(text string) {
		field := strings.Split(text, "\t")
		rows = append(rows, &Row{
			commit: field[0],
			title:  field[1],
		})
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func main1() error {
	rows, err := makeRows()
	if err != nil {
		return err
	}
	statusLine := "  [j] Up  [k] Down  [SPACE] git show  [q] Quit"
	return twopane.View{Rows: rows, StatusLine: statusLine}.Run()
}

func main() {
	if err := main1(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
