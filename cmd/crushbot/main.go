package main

import (
	"os"

	"github.com/dukedelaet/crush-bot/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
