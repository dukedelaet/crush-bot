package cli

import (
	"fmt"

	"github.com/hocoder-agents/crush-bot/internal/sandbox"
)

func cmdSandboxExec(io IO, args []string) int {
	bin, root, bot := "", "", ""
	var crushArgs []string
	dash := false
	for i := 0; i < len(args); i++ {
		if dash {
			crushArgs = append(crushArgs, args[i])
			continue
		}
		switch args[i] {
		case "--":
			dash = true
		case "--bin":
			i++
			if i < len(args) {
				bin = args[i]
			}
		case "--root":
			i++
			if i < len(args) {
				root = args[i]
			}
		case "--bot":
			i++
			if i < len(args) {
				bot = args[i]
			}
		default:
			fmt.Fprintln(io.Err, errStyle.Render("usage: crushbot sandbox-exec --bin <crush> --root <home> --bot <slug> -- <crush args>"))
			return 2
		}
	}
	if bin == "" || root == "" || bot == "" {
		fmt.Fprintln(io.Err, errStyle.Render("sandbox-exec missing --bin/--root/--bot"))
		return 2
	}
	if err := sandbox.ExecLandlocked(bin, root, bot, crushArgs); err != nil {
		fmt.Fprintln(io.Err, errStyle.Render(err.Error()))
		return 1
	}
	return 0
}
