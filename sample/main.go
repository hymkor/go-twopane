package main

import (
	"fmt"
	"os"

	"github.com/zetamatta/go-twopane"
)

type Row string

func (r Row) Title() string {
	return string(r)
}

func (r Row) Contents() []string {
	return []string{"Contents of " + string(r)}
}

func main() {
	rows := []twopane.Row{}
	for i := 0; i < 100; i++ {
		rows = append(rows, Row(fmt.Sprintf("%d", i)))
	}
	err := twopane.View{Rows: rows}.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
