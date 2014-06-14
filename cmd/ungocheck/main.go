package main

import (
	"fmt"
	"os"

	"github.com/rjeczalik/ungocheck"
)

func die(v interface{}) {
	fmt.Fprintln(os.Stderr, v)
	os.Exit(1)
}

func main() {
	if err := ungocheck.New().Run(os.Args); err != nil {
		die(err)
	}
}
