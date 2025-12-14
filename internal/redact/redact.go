package redact

import (
	"regexp"
	"strings"
)

type Redactor struct {
	secretLike []*regexp.Regexp
	flagValue  map[string]bool
}

func Default() *Redactor {
	return &Redactor{
		secretLike: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\bghp_[A-Za-z0-9]{20,}\b`),
			regexp.MustCompile(`(?i)\bgithub_pat_[A-Za-z0-9_]{20,}\b`),
			regexp.MustCompile(`(?i)\bAKIA[0-9A-Z]{16}\b`),
			regexp.MustCompile(`(?i)\bxox[baprs]-[A-Za-z0-9-]{10,}\b`),
			regexp.MustCompile(`(?i)\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`),
			regexp.MustCompile(`(?i)\bsk-[A-Za-z0-9]{20,}\b`),     // OpenAI-style keys
			regexp.MustCompile(`(?i)\bAIza[0-9A-Za-z_-]{20,}\b`),  // Google API keys
			regexp.MustCompile(`(?i)https?://[^/\s:]+:[^@\s/]+@`), // URL basic auth creds
			regexp.MustCompile(`(?i)authorization:\s*(?:bearer|token)\s+[A-Za-z0-9._~+/=-]{8,}`),
		},
		flagValue: map[string]bool{
			"--token": true, "--password": true, "--pass": true,
			"--apikey": true, "--api-key": true,
		},
	}
}

func (r *Redactor) RedactArgs(argv []string) []string {
	out := make([]string, 0, len(argv))
	skipNext := false
	for i, a := range argv {
		if skipNext {
			out = append(out, "<redacted>")
			skipNext = false
			continue
		}
		if k, _, ok := strings.Cut(a, "="); ok && r.flagValue[k] {
			out = append(out, k+"=<redacted>")
			continue
		}
		if r.flagValue[a] && i+1 < len(argv) {
			out = append(out, a)
			skipNext = true
			continue
		}
		out = append(out, r.RedactText(a))
	}
	return out
}

func (r *Redactor) RedactText(s string) string {
	out := s
	for _, re := range r.secretLike {
		out = re.ReplaceAllString(out, "<redacted>")
	}
	return out
}
