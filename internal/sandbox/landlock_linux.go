//go:build linux

package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/dukedelaet/crush-bot/internal/roster"
)

const (
	sysLandlockCreateRuleset = 444
	sysLandlockAddRule       = 445
	sysLandlockRestrictSelf  = 446

	landlockRulePathBeneath      = 1
	landlockCreateRulesetVersion = 1

	accessFSExecute    = 1 << 0
	accessFSWriteFile  = 1 << 1
	accessFSReadFile   = 1 << 2
	accessFSReadDir    = 1 << 3
	accessFSRemoveDir  = 1 << 4
	accessFSRemoveFile = 1 << 5
	accessFSMakeChar   = 1 << 6
	accessFSMakeDir    = 1 << 7
	accessFSMakeReg    = 1 << 8
	accessFSMakeSock   = 1 << 9
	accessFSMakeFIFO   = 1 << 10
	accessFSMakeBlock  = 1 << 11
	accessFSMakeSym    = 1 << 12
)

const handledFS = accessFSExecute | accessFSWriteFile | accessFSReadFile | accessFSReadDir |
	accessFSRemoveDir | accessFSRemoveFile | accessFSMakeChar | accessFSMakeDir |
	accessFSMakeReg | accessFSMakeSock | accessFSMakeFIFO | accessFSMakeBlock | accessFSMakeSym

const accessRO = accessFSExecute | accessFSReadFile | accessFSReadDir
const accessRW = handledFS

type rulesetAttr struct {
	HandledAccessFS uint64
}

type pathBeneathAttr struct {
	AllowedAccess uint64
	ParentFd      int32
}

func landlockAvailable() error {
	r1, _, errno := unix.Syscall(sysLandlockCreateRuleset, 0, 0, landlockCreateRulesetVersion)
	if errno != 0 {
		return fmt.Errorf("landlock not available: %v", errno)
	}
	if int(r1) < 1 {
		return fmt.Errorf("landlock ABI %d too old", int(r1))
	}
	return nil
}

func applyLandlock(bot roster.Bot, root, crushBin, self string) error {
	attr := rulesetAttr{HandledAccessFS: handledFS}
	r1, _, errno := unix.Syscall(sysLandlockCreateRuleset, uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr), 0)
	if errno != 0 {
		return fmt.Errorf("landlock_create_ruleset: %v", errno)
	}
	fd := int(r1)
	defer unix.Close(fd)

	ro := []string{"/usr", "/bin", "/lib", "/lib64", "/etc/ssl", "/etc/resolv.conf", "/proc", "/dev", crushBin, xdgConfigHome() + "/crush", root}
	if self != "" {
		ro = append(ro, self)
	}
	home := roster.Home(root, bot.Slug)
	rw := []string{home, "/tmp", filepath.Join(root, "needs_you.jsonl")}
	if bot.Project != "" {
		rw = append(rw, bot.Project)
	}
	bots, _ := roster.List(root, true)
	for _, b := range bots {
		if b.Slug == bot.Slug {
			continue
		}
		rw = append(rw, filepath.Join(roster.Home(root, b.Slug), "inbox", "pending"))
	}
	for _, p := range ro {
		_ = addPath(fd, p, accessRO)
	}
	for _, p := range rw {
		if err := addPath(fd, p, accessRW); err != nil {
			return err
		}
	}
	if _, _, errno = unix.Syscall6(syscall.SYS_PRCTL, unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0, 0); errno != 0 {
		return fmt.Errorf("PR_SET_NO_NEW_PRIVS: %v", errno)
	}
	if _, _, errno = unix.Syscall(sysLandlockRestrictSelf, uintptr(fd), 0, 0); errno != 0 {
		return fmt.Errorf("landlock_restrict_self: %v", errno)
	}
	return nil
}

func addPath(ruleset int, path string, access uint64) error {
	if path == "" {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	f, err := os.OpenFile(path, unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil
	}
	defer f.Close()
	pb := pathBeneathAttr{AllowedAccess: access, ParentFd: int32(f.Fd())}
	_, _, errno := unix.Syscall6(sysLandlockAddRule, uintptr(ruleset), landlockRulePathBeneath, uintptr(unsafe.Pointer(&pb)), 0, 0, 0)
	if errno != 0 {
		return fmt.Errorf("landlock add %s: %v", path, errno)
	}
	return nil
}

func landlockExecCmd(crushbot, crushBin string, crushArgs []string, bot roster.Bot, root string) (string, []string) {
	args := []string{"sandbox-exec", "--bin", crushBin, "--root", root, "--bot", bot.Slug, "--"}
	args = append(args, crushArgs...)
	return crushbot, args
}
