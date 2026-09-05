package daemon

import "strings"

func Classify(exit int, stderr, stdout string) (reason string, retryable bool) {
	if exit == 0 {
		return "", false
	}
	blob := strings.ToLower(stderr + "\n" + stdout)
	switch {
	case strings.Contains(blob, "rate limit"), strings.Contains(blob, "too many requests"):
		return "provider_rate_limit", true
	case strings.Contains(blob, "context overflow"), strings.Contains(blob, "context length"):
		return "context_overflow", true
	case strings.Contains(blob, "timeout"), strings.Contains(blob, "deadline exceeded"):
		return "delivery_timeout", true
	case strings.Contains(blob, "500"), strings.Contains(blob, "502"), strings.Contains(blob, "503"),
		strings.Contains(blob, "internal server"):
		return "provider_server_error", true
	case strings.Contains(blob, "401"), strings.Contains(blob, "403"), strings.Contains(blob, "unauthorized"),
		strings.Contains(blob, "invalid api key"):
		return "provider_auth_or_access", false
	case strings.Contains(blob, "quota"):
		return "provider_quota_limit", false
	case strings.Contains(blob, "no providers configured"):
		return "missing_config", false
	default:
		return "unknown", false
	}
}
