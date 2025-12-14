package redact

import "testing"

func BenchmarkRedactText(b *testing.B) {
	r := Default()
	s := "Authorization: Bearer sk-0123456789abcdefghijklmnopqrstuvwxyz token=ghp_0123456789ABCDEFGHijklmnopqrs https://user:pass@example.com/path"

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = r.RedactText(s)
	}
}

func BenchmarkRedactArgs(b *testing.B) {
	r := Default()
	argv := []string{
		"curl",
		"-H", "Authorization: Bearer sk-0123456789abcdefghijklmnopqrstuvwxyz",
		"--token=ghp_0123456789ABCDEFGHijklmnopqrs",
		"--api-key", "AIzaSyA_0123456789abcdefGHIJKL",
		"https://user:pass@example.com/path?token=ghp_0123456789ABCDEFGHijklmnopqrs",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = r.RedactArgs(argv)
	}
}
