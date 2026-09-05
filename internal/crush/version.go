package crush

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var versionRe = regexp.MustCompile(`v?(\d+)\.(\d+)\.(\d+)`)

const MinVersion = "0.91.2"

func ParseVersion(s string) (maj, min, patch int, ok bool) {
	m := versionRe.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, 0, false
	}
	maj, _ = strconv.Atoi(m[1])
	min, _ = strconv.Atoi(m[2])
	patch, _ = strconv.Atoi(m[3])
	return maj, min, patch, true
}

func AtLeast(have, want string) bool {
	h1, h2, h3, okh := ParseVersion(have)
	w1, w2, w3, okw := ParseVersion(want)
	if !okh || !okw {
		return false
	}
	if h1 != w1 {
		return h1 > w1
	}
	if h2 != w2 {
		return h2 > w2
	}
	return h3 >= w3
}

func Version(bin string) (string, error) {
	out, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("crush --version: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	s := strings.TrimSpace(string(out))
	if _, _, _, ok := ParseVersion(s); !ok {
		return "", fmt.Errorf("unparseable crush version: %q", s)
	}
	return s, nil
}

func RequireMin(bin, min string) error {
	v, err := Version(bin)
	if err != nil {
		return err
	}
	if !AtLeast(v, min) {
		return fmt.Errorf("crush %s is older than required %s", v, min)
	}
	return nil
}

func HasRootYolo(bin string) bool {
	out, err := exec.Command(bin, "--help").CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "--yolo")
}
