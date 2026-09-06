package cli

import (
	"fmt"
	"os"

	"github.com/hocoder-agents/crush-bot/internal/mesh"
)

func cmdMCP(io IO, _ []string) int {
	id := mesh.IdentityFromEnv()
	if err := mesh.Serve(os.Stdin, os.Stdout, id); err != nil {
		fmt.Fprintln(io.Err, err.Error())
		return 1
	}
	return 0
}
