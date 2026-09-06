package main

import (
	"os"

	"github.com/hocoder-agents/crush-bot/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
