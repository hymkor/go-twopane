package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/zetamatta/twopainview"
)

type Commit struct {
	commit   string
	title    string
	contents []string
}

func (this *Commit) Title() string {
	return this.title
}

func (this *Commit) Contents() []string {
	if this.contents == nil {
		cmd := exec.Command("git", "show", this.commit)
		in, err := cmd.StdoutPipe()
		if err != nil {
			this.contents = []string{err.Error()}
			return this.contents
		}
		err = cmd.Start()
		if err != nil {
			this.contents = []string{err.Error()}
			return this.contents
		}
		sc := bufio.NewScanner(in)
		for sc.Scan() {
			this.contents = append(this.contents, sc.Text())
		}
		cmd.Wait()
		in.Close()
	}
	return this.contents
}

func main1() error {
	commits := []twopainview.Node{}
	cmd := exec.Command("git", "log", "-n", "100", "--pretty=format:%H\t%h %s")

	in, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	sc := bufio.NewScanner(in)
	for sc.Scan() {
		text := sc.Text()
		field := strings.Split(text, "\t")
		commits = append(commits, &Commit{
			commit: field[0],
			title:  field[1],
		})
	}
	cmd.Wait()
	in.Close()
	return twopainview.Main(commits, 15)
}

func main() {
	if err := main1(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
