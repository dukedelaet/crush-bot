package crush

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type InboundRef struct {
	ID     string   `json:"id"`
	Kind   string   `json:"kind"`
	From   string   `json:"from"`
	Hop    int      `json:"hop"`
	Trace  []string `json:"trace"`
	TaskID *string  `json:"task_id"`
}

type Turn struct {
	Bot           string       `json:"bot"`
	SessionID     string       `json:"session_id"`
	Kind          string       `json:"kind"`
	StartedAt     time.Time    `json:"started_at"`
	EndedAt       *time.Time   `json:"ended_at,omitempty"`
	CrushPID      int          `json:"crush_pid"`
	Inbound       []InboundRef `json:"inbound"`
	InboundHop    int          `json:"inbound_hop"`
	Trace         []string     `json:"trace"`
	ParentID      string       `json:"parent_id,omitempty"`
	Sends         int          `json:"sends"`
	MaxSends      int          `json:"max_sends"`
	MaxHops       int          `json:"max_hops"`
	GroupID       *string      `json:"group_id"`
	GroupSends    int          `json:"group_sends"`
	MaxGroupSends int          `json:"max_group_sends"`
}

func TurnPath(botHome string) string {
	return filepath.Join(botHome, "turn.json")
}

func LockPath(botHome string) string {
	return filepath.Join(botHome, "turn.lock")
}

func MaxSendsFor(kind string) int {
	if kind == "human_chat" {
		return 32
	}
	return 4
}

func WriteTurn(botHome string, t Turn) error {
	t.CrushPID = 0
	if t.StartedAt.IsZero() {
		t.StartedAt = time.Now().UTC()
	}
	if t.MaxSends == 0 {
		t.MaxSends = MaxSendsFor(t.Kind)
	}
	if t.MaxHops == 0 {
		t.MaxHops = 8
	}
	if t.MaxGroupSends == 0 {
		t.MaxGroupSends = 10
	}
	if t.Trace == nil {
		t.Trace = []string{}
	}
	if t.Inbound == nil {
		t.Inbound = []InboundRef{}
	}
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(TurnPath(botHome), b, 0o600)
}

func ReadTurn(botHome string) (Turn, error) {
	b, err := os.ReadFile(TurnPath(botHome))
	if err != nil {
		return Turn{}, err
	}
	var t Turn
	if err := json.Unmarshal(b, &t); err != nil {
		return Turn{}, err
	}
	return t, nil
}

func UpdatePID(botHome string, pid int) error {
	path := TurnPath(botHome)
	f, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("fcntl turn.json: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	var t Turn
	if err := json.NewDecoder(f).Decode(&t); err != nil {
		return err
	}
	t.CrushPID = pid
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	if err := f.Truncate(0); err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(t)
}

func RemoveTurn(botHome string) {
	_ = os.Remove(TurnPath(botHome))
}

func PIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil
}
