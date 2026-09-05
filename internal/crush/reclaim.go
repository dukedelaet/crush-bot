package crush

import (
	"os"
	"path/filepath"
	"time"
)

const pidZeroGrace = 2 * time.Second

// ReclaimStale releases a dead turn: processing/ → pending (attempt unchanged).
func ReclaimStale(botHome string) (reclaimed bool, err error) {
	t, err := ReadTurn(botHome)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if t.CrushPID == 0 {
		time.Sleep(pidZeroGrace)
		t, err = ReadTurn(botHome)
		if os.IsNotExist(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if t.CrushPID != 0 && PIDAlive(t.CrushPID) {
			return false, nil
		}
	} else if PIDAlive(t.CrushPID) {
		return false, nil
	}
	if err := moveProcessingToPending(botHome); err != nil {
		return false, err
	}
	RemoveTurn(botHome)
	return true, nil
}

func moveProcessingToPending(botHome string) error {
	proc := filepath.Join(botHome, "inbox", "processing")
	pend := filepath.Join(botHome, "inbox", "pending")
	if err := os.MkdirAll(pend, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(proc)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(proc, e.Name())
		dst := filepath.Join(pend, e.Name())
		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}
	return nil
}
