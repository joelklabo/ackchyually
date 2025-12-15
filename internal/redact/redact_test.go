package redact

import (
	"strings"
	"testing"
)

func TestRedactArgs_FlagValueForms(t *testing.T) {
	r := Default()

	got := r.RedactArgs([]string{"gh", "auth", "login", "--token", "secret"})
	if got[3] != "--token" || got[4] != "<redacted>" {
		t.Fatalf("RedactArgs(--token secret)=%#v", got)
	}

	got = r.RedactArgs([]string{"gh", "auth", "login", "--token=secret"})
	if got[3] != "--token=<redacted>" {
		t.Fatalf("RedactArgs(--token=secret)=%#v", got)
	}
}

func TestRedactArgs_SecretLikePatterns(t *testing.T) {
	r := Default()

	url := "https://user:pass@example.com/path"
	got := r.RedactArgs([]string{"curl", url})
	if strings.Contains(strings.Join(got, " "), "user:pass@") {
		t.Fatalf("expected URL creds to be redacted, got %#v", got)
	}
	if !strings.Contains(strings.Join(got, " "), "<redacted>") {
		t.Fatalf("expected <redacted> to appear, got %#v", got)
	}

	auth := "Authorization: Bearer abcdefghijklmnop"
	got = r.RedactArgs([]string{auth})
	if len(got) != 1 || got[0] != "<redacted>" {
		t.Fatalf("expected authorization header to be redacted, got %#v", got)
	}
}
