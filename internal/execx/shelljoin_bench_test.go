package execx

import "testing"

func BenchmarkShellJoin(b *testing.B) {
	argv := []string{
		"git",
		"commit",
		"-m",
		"fix: handle spaces and 'quotes'",
		"--author=Bob Example <bob@example.com>",
		"--path=/Users/alice/projects/repo",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ShellJoin(argv)
	}
}
