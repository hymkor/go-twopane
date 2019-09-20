// +build ignore

package main

import (
	"fmt"
	"os"

	"github.com/zetamatta/twopainview"
)

type nodeT string

func (this nodeT) Title() string {
	return string(this)
}

func (this nodeT) Contents() []string {
	contents := make([]string, 0, 30)
	for i := 0; i < 30; i++ {
		contents = append(contents, fmt.Sprintf("%s - %d", string(this), i))
	}
	return contents
}

func main() {
	nodes := []twopainview.Node{}

	for i := 0; i < 100; i++ {
		nodes = append(nodes, nodeT(fmt.Sprintf("[%05d]", i)))
	}

	err := twopainview.Main(nodes)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
