package store

import "testing"

func BenchmarkMustJSON_Argv(b *testing.B) {
	argv := []string{"git", "status", "--porcelain"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = MustJSON(argv)
	}
}
