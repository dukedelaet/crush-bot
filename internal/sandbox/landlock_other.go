//go:build !linux

package sandbox

import (
	"fmt"

	"github.com/dukedelaet/crush-bot/internal/roster"
)

func landlockAvailable() error {
	return fmt.Errorf("landlock is Linux-only")
}

func applyLandlock(bot roster.Bot, root, crushBin, self string) error {
	return fmt.Errorf("landlock is Linux-only")
}

func landlockExecCmd(crushbot, crushBin string, crushArgs []string, bot roster.Bot, root string) (string, []string) {
	return crushBin, crushArgs
}
