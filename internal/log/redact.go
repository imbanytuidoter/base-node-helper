package log

import "regexp"

var redactPatterns = []struct {
	re   *regexp.Regexp
	repl string
}{
	{regexp.MustCompile(`(/v[23]/)[A-Za-z0-9_\-]{20,}`), `${1}****`},
	{regexp.MustCompile(`/[a-f0-9]{32,}/`), `/****/`},
	{regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9_\-\.=]{16,}`), `${1}****`},
	{regexp.MustCompile(`(?i)([?&](api[_-]?key|key|token)=)[^&\s#]+`), `${1}****`},
	// [INFO] password group uses * (not +) to also redact empty passwords (user:@host)
	{regexp.MustCompile(`(?i)(://)[^:@/\s]+:[^@/\s]*@`), `${1}****:****@`},
	{regexp.MustCompile(`(?i)(wss?://[^/]+/)[A-Za-z0-9_\-]{20,}`), `${1}****`},
	{regexp.MustCompile(`(?i)([?&](project[-_]?id|access[-_]?key|secret[-_]?key|auth[-_]?token)=)[^&\s#]+`), `${1}****`},
}

// Redact masks secret-looking substrings in s. Idempotent.
func Redact(s string) string {
	for _, p := range redactPatterns {
		s = p.re.ReplaceAllString(s, p.repl)
	}
	return s
}
