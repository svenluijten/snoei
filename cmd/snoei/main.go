package main

import (
	"os"

	"github.com/svenluijten/snoei/internal/cli"
	"github.com/svenluijten/snoei/internal/picker"
	"golang.org/x/term"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Env{
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Interactive: term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())),
		Pick:        picker.Pick,
	}))
}
