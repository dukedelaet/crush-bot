package soul

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

const DefaultMaxBytes = 32768

func SeedMarkdown(slug string) string {
	return fmt.Sprintf(`# Identity
You are %s, a Crush-backed specialist bot.

# Style
Be direct. Match reply length to the weight of the ask.
No filler, no restating the request.

# Avoid
Sycophancy. Hype. Narrating tool calls the operator can already see.

# Defaults
If a teammate is a better fit, message them or assign a task instead of stretching.
`, slug)
}

// WriteSeed writes soul.md once. Existing files are left untouched.
func WriteSeed(path, slug string) (created bool, err error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	return true, os.WriteFile(path, []byte(SeedMarkdown(slug)), 0o600)
}

func Read(path string, maxBytes int) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	if len(b) > maxBytes {
		b = b[:maxBytes]
		for !utf8.Valid(b) && len(b) > 0 {
			b = b[:len(b)-1]
		}
	}
	return string(b), nil
}

func SHA256(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

func IsBlank(body string) bool {
	return strings.TrimSpace(body) == ""
}
